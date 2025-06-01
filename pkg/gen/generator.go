package gen

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed templates/*.tmpl
var defaultTemplates embed.FS

// Config holds the configuration for code generation
type Config struct {
	// SpecPath is the path to the OpenAPI specification file
	SpecPath string
	// OutputDir is the directory where generated files will be written
	OutputDir string
	// PackageName is the name of the generated Go package
	PackageName string
	// ClientName is the name of the generated client struct
	ClientName string
	// TemplateDir is an optional directory containing custom templates
	TemplateDir string
	// ModelPackage is the package name for models (if different from main package)
	ModelPackage string
	// ClientPackage is the package name for client (if different from main package)
	ClientPackage string
	// ClientImport is the custom import path for client packages
	ClientImport string
	// GenerateModels indicates whether to generate model files
	GenerateModels bool
	// GenerateClient indicates whether to generate client files
	GenerateClient bool
	// EmbedClient indicates whether to copy client packages instead of importing
	EmbedClient bool
	// Verbose enables verbose output
	Verbose bool
}

// Generator generates Go code from OpenAPI specifications
type Generator struct {
	config    *Config
	spec      *openapi3.T
	templates *template.Template
}

// NewGenerator creates a new code generator
func NewGenerator(config *Config) (*Generator, error) {
	if config.PackageName == "" {
		return nil, fmt.Errorf("package name is required")
	}
	if config.ClientName == "" {
		config.ClientName = "Client"
	}
	if config.ModelPackage == "" {
		config.ModelPackage = config.PackageName
	}
	if config.ClientPackage == "" {
		config.ClientPackage = config.PackageName
	}

	gen := &Generator{
		config: config,
	}

	// Load templates
	if err := gen.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return gen, nil
}

// loadTemplates loads templates from embedded FS or custom directory
func (g *Generator) loadTemplates() error {
	tmpl := template.New("").Funcs(templateFuncs())

	// Load from custom directory if specified
	if g.config.TemplateDir != "" {
		pattern := filepath.Join(g.config.TemplateDir, "*.tmpl")
		var err error
		tmpl, err = tmpl.ParseGlob(pattern)
		if err != nil {
			return fmt.Errorf("failed to parse custom templates: %w", err)
		}
	} else {
		// Load embedded templates
		err := fs.WalkDir(defaultTemplates, "templates", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
				return nil
			}

			content, err := defaultTemplates.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read template %s: %w", path, err)
			}

			name := strings.TrimPrefix(path, "templates/")
			name = strings.TrimSuffix(name, ".tmpl")

			_, err = tmpl.New(name).Parse(string(content))
			if err != nil {
				return fmt.Errorf("failed to parse template %s: %w", name, err)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to load embedded templates: %w", err)
		}
	}

	g.templates = tmpl
	return nil
}

// LoadSpec loads and validates the OpenAPI specification
func (g *Generator) LoadSpec() error {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	spec, err := loader.LoadFromFile(g.config.SpecPath)
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	// Validate the spec
	ctx := loader.Context
	if err := spec.Validate(ctx); err != nil {
		return fmt.Errorf("invalid OpenAPI spec: %w", err)
	}

	g.spec = spec
	return nil
}

// Generate generates Go code from the loaded specification
func (g *Generator) Generate() error {
	if g.spec == nil {
		return fmt.Errorf("specification not loaded")
	}

	// Create output directory
	if err := os.MkdirAll(g.config.OutputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate models
	if g.config.GenerateModels {
		if err := g.generateModels(); err != nil {
			return fmt.Errorf("failed to generate models: %w", err)
		}
	}

	// Generate client
	if g.config.GenerateClient {
		if err := g.generateClient(); err != nil {
			return fmt.Errorf("failed to generate client: %w", err)
		}
	}

	return nil
}

// generateModels generates model files from OpenAPI schemas
func (g *Generator) generateModels() error {
	// Prepare model data
	models := g.extractModels()

	data := map[string]interface{}{
		"Package": g.config.ModelPackage,
		"Models":  models,
		"Imports": g.getModelImports(models),
	}

	// Generate models file
	outputPath := filepath.Join(g.config.OutputDir, "models.go")
	if g.config.ModelPackage != g.config.PackageName {
		// Create subdirectory for models
		modelDir := filepath.Join(g.config.OutputDir, "models")
		if err := os.MkdirAll(modelDir, 0o755); err != nil {
			return fmt.Errorf("failed to create models directory: %w", err)
		}
		outputPath = filepath.Join(modelDir, "models.go")
	}

	return g.generateFile("models", data, outputPath)
}

// generateClient generates client files from OpenAPI paths
func (g *Generator) generateClient() error {
	// Prepare client data
	operations := g.extractOperations()

	data := map[string]interface{}{
		"Package":      g.config.ClientPackage,
		"ClientName":   g.config.ClientName,
		"Operations":   operations,
		"Imports":      g.getClientImports(operations),
		"ModelPackage": g.config.ModelPackage,
		"BaseURL":      g.getBaseURL(),
	}

	// Generate client file
	outputPath := filepath.Join(g.config.OutputDir, fmt.Sprintf("%s.go", toSnakeCase(g.config.ClientName)))

	return g.generateFile("client", data, outputPath)
}

// generateFile generates a single file from a template
func (g *Generator) generateFile(templateName string, data interface{}, outputPath string) error {
	if g.config.Verbose {
		fmt.Printf("Generating %s...\n", outputPath)
	}

	// Execute template
	var buf bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted code for debugging
		if g.config.Verbose {
			fmt.Printf("Warning: failed to format generated code: %v\n", err)
			fmt.Printf("Writing unformatted code to %s.unformatted\n", outputPath)
			_ = os.WriteFile(outputPath+".unformatted", buf.Bytes(), 0o644)
		}
		return fmt.Errorf("failed to format generated code: %w", err)
	}

	// Write the file
	if err := os.WriteFile(outputPath, formatted, 0o644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", outputPath, err)
	}

	return nil
}

// Model represents a Go model generated from an OpenAPI schema
type Model struct {
	Name        string
	Description string
	Fields      []Field
	IsEnum      bool
	EnumValues  []string
}

// Field represents a field in a model
type Field struct {
	Name        string
	JSONName    string
	Type        string
	Description string
	Required    bool
	Nullable    bool
	OmitEmpty   bool
}

// Operation represents an API operation
type Operation struct {
	Name                        string
	Method                      string
	Path                        string
	Description                 string
	OperationID                 string
	Parameters                  []Parameter
	RequestBody                 *RequestBody
	Responses                   map[string]Response
	SuccessResponse             *Response
	HasMultipleSuccessResponses bool
	ErrorResponses              []Response
}

// Parameter represents an API parameter
type Parameter struct {
	Name        string
	In          string // path, query, header, cookie
	Type        string
	Description string
	Required    bool
}

// RequestBody represents a request body
type RequestBody struct {
	Type        string
	Description string
	Required    bool
}

// Response represents an API response
type Response struct {
	StatusCode  string
	Type        string
	Description string
}

// extractModels extracts model definitions from the OpenAPI spec
func (g *Generator) extractModels() []Model {
	var models []Model

	if g.spec.Components == nil || g.spec.Components.Schemas == nil {
		return models
	}

	for name, schemaRef := range g.spec.Components.Schemas {
		if schemaRef.Value == nil {
			continue
		}

		model := g.schemaToModel(name, schemaRef.Value)
		if model != nil {
			models = append(models, *model)
		}
	}

	return models
}

// schemaToModel converts an OpenAPI schema to a Model
func (g *Generator) schemaToModel(name string, schema *openapi3.Schema) *Model {
	model := &Model{
		Name:        toPascalCase(name),
		Description: schema.Description,
	}

	// Handle enums
	if len(schema.Enum) > 0 {
		model.IsEnum = true
		for _, v := range schema.Enum {
			if str, ok := v.(string); ok {
				model.EnumValues = append(model.EnumValues, str)
			}
		}
		return model
	}

	// Handle object types
	if schema.Type != nil && schema.Type.Is("object") {
		required := make(map[string]bool)
		for _, r := range schema.Required {
			required[r] = true
		}

		for propName, propRef := range schema.Properties {
			if propRef.Value == nil {
				continue
			}

			field := Field{
				Name:        toPascalCase(propName),
				JSONName:    propName,
				Type:        g.schemaRefToGoType(propRef),
				Description: propRef.Value.Description,
				Required:    required[propName],
				Nullable:    propRef.Value.Nullable,
				OmitEmpty:   !required[propName],
			}

			model.Fields = append(model.Fields, field)
		}
	}

	return model
}

// schemaToGoType converts an OpenAPI schema to a Go type
func (g *Generator) schemaToGoType(schema *openapi3.Schema) string {
	if schema == nil {
		return "interface{}"
	}

	// Handle references
	if ref := getSchemaRef(schema); ref != "" {
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return toPascalCase(parts[len(parts)-1])
		}
	}

	// Handle arrays
	if schema.Type != nil && schema.Type.Is("array") {
		if schema.Items != nil {
			return "[]" + g.schemaRefToGoType(schema.Items)
		}
		return "[]interface{}"
	}

	// Handle basic types
	if schema.Type != nil {
		typeStr := ""
		if len(*schema.Type) > 0 {
			typeStr = (*schema.Type)[0]
		}

		switch typeStr {
		case "string":
			if schema.Format == "date-time" {
				return "time.Time"
			}
			if schema.Format == "date" {
				return "time.Time"
			}
			return "string"
		case "integer":
			if schema.Format == "int32" {
				return "int32"
			}
			return "int64"
		case "number":
			if schema.Format == "float" {
				return "float32"
			}
			return "float64"
		case "boolean":
			return "bool"
		case "object":
			// Check if it has additionalProperties defined
			if schema.AdditionalProperties.Schema != nil {
				if schema.AdditionalProperties.Schema.Value != nil {
					return "map[string]" + g.schemaToGoType(schema.AdditionalProperties.Schema.Value)
				}
				// Handle case where additionalProperties is a reference
				return "map[string]" + g.schemaRefToGoType(schema.AdditionalProperties.Schema)
			}
			// If Has is explicitly set to true, it's a map[string]interface{}
			if schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
				return "map[string]interface{}"
			}
			return "interface{}"
		}
	}

	return "interface{}"
}

// extractOperations extracts operations from the OpenAPI spec
func (g *Generator) extractOperations() []Operation {
	var operations []Operation

	if g.spec.Paths != nil {
		// Use InMatchingOrder() to preserve path order
		for _, path := range g.spec.Paths.InMatchingOrder() {
			pathItem := g.spec.Paths.Value(path)
			if pathItem != nil {
				operations = append(operations, g.extractPathOperations(path, pathItem)...)
			}
		}
	}

	return operations
}

// schemaRefToGoType converts an OpenAPI schema reference to a Go type
func (g *Generator) schemaRefToGoType(schemaRef *openapi3.SchemaRef) string {
	if schemaRef == nil {
		return "interface{}"
	}

	// Check if this is a reference
	if schemaRef.Ref != "" {
		// Extract the type name from the reference
		// e.g., "#/components/schemas/User" -> "User"
		parts := strings.Split(schemaRef.Ref, "/")
		if len(parts) > 0 {
			return toPascalCase(parts[len(parts)-1])
		}
	}

	// Otherwise, process the schema value
	if schemaRef.Value != nil {
		return g.schemaToGoType(schemaRef.Value)
	}

	return "interface{}"
}

// extractPathOperations extracts operations from a path item
func (g *Generator) extractPathOperations(path string, pathItem *openapi3.PathItem) []Operation {
	var operations []Operation

	// Helper function to process an operation
	processOp := func(method string, op *openapi3.Operation) {
		if op == nil {
			return
		}

		operation := Operation{
			Method:      method,
			Path:        path,
			Description: op.Description,
			OperationID: op.OperationID,
			Responses:   make(map[string]Response),
		}

		// Generate operation name
		if op.OperationID != "" {
			operation.Name = toPascalCase(op.OperationID)
		} else {
			operation.Name = generateOperationName(method, path)
		}

		// Extract parameters
		for _, paramRef := range op.Parameters {
			if paramRef.Value == nil {
				continue
			}
			param := Parameter{
				Name:        paramRef.Value.Name,
				In:          paramRef.Value.In,
				Description: paramRef.Value.Description,
				Required:    paramRef.Value.Required,
			}
			if paramRef.Value.Schema != nil {
				param.Type = g.schemaRefToGoType(paramRef.Value.Schema)
			}
			operation.Parameters = append(operation.Parameters, param)
		}

		// Extract request body
		if op.RequestBody != nil && op.RequestBody.Value != nil {
			rb := op.RequestBody.Value
			if content, ok := rb.Content["application/json"]; ok && content.Schema != nil {
				operation.RequestBody = &RequestBody{
					Type:        g.schemaRefToGoType(content.Schema),
					Description: rb.Description,
					Required:    rb.Required,
				}
			}
		}

		// Extract responses
		if op.Responses != nil {
			successCount := 0
			for statusCode, responseRef := range op.Responses.Map() {
				if responseRef.Value == nil {
					continue
				}
				desc := ""
				if responseRef.Value.Description != nil {
					desc = *responseRef.Value.Description
				}
				resp := Response{
					StatusCode:  statusCode,
					Description: desc,
				}
				if content, ok := responseRef.Value.Content["application/json"]; ok && content.Schema != nil {
					resp.Type = g.schemaRefToGoType(content.Schema)
				} else if content, ok := responseRef.Value.Content["*/*"]; ok && content.Schema != nil {
					// Handle wildcard content type
					resp.Type = g.schemaRefToGoType(content.Schema)
				}
				operation.Responses[statusCode] = resp

				// Track success responses (2xx)
				if strings.HasPrefix(statusCode, "2") {
					successCount++
					if operation.SuccessResponse == nil {
						operation.SuccessResponse = &resp
					}
				}

				// Track error responses (4xx and 5xx)
				if strings.HasPrefix(statusCode, "4") || strings.HasPrefix(statusCode, "5") {
					operation.ErrorResponses = append(operation.ErrorResponses, resp)
				}
			}
			operation.HasMultipleSuccessResponses = successCount > 1
		}

		operations = append(operations, operation)
	}

	// Process all HTTP methods
	processOp("GET", pathItem.Get)
	processOp("POST", pathItem.Post)
	processOp("PUT", pathItem.Put)
	processOp("DELETE", pathItem.Delete)
	processOp("PATCH", pathItem.Patch)
	processOp("HEAD", pathItem.Head)
	processOp("OPTIONS", pathItem.Options)

	return operations
}

// getModelImports returns required imports for models
func (g *Generator) getModelImports(models []Model) []string {
	imports := make(map[string]bool)

	for _, model := range models {
		for _, field := range model.Fields {
			if strings.Contains(field.Type, "time.Time") {
				imports["time"] = true
			}
		}
	}

	var result []string
	for imp := range imports {
		result = append(result, imp)
	}
	return result
}

// getClientImports returns required imports for client
func (g *Generator) getClientImports(operations []Operation) []string {
	// Use custom client import path if specified, otherwise use default
	clientImportPath := g.config.ClientImport
	if clientImportPath == "" {
		clientImportPath = "github.com/jmcarbo/oapix/pkg/client"
	}

	imports := map[string]bool{
		"context":        true,
		"fmt":            true,
		clientImportPath: true,
	}

	// Add model import if needed
	if g.config.ModelPackage != g.config.ClientPackage {
		imports[fmt.Sprintf("github.com/jmcarbo/oapix/%s/models", g.config.OutputDir)] = true
	}

	var result []string
	for imp := range imports {
		result = append(result, imp)
	}
	return result
}

// getBaseURL extracts base URL from the spec
func (g *Generator) getBaseURL() string {
	if len(g.spec.Servers) > 0 {
		return g.spec.Servers[0].URL
	}
	return "https://api.example.com"
}

// GenerateFromReader generates code from an OpenAPI spec reader
func GenerateFromReader(reader io.Reader, config *Config) error {
	// Create temporary file
	tmpFile, err := os.CreateTemp("", "openapi-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	// Copy reader to temp file
	if _, err := io.Copy(tmpFile, reader); err != nil {
		return fmt.Errorf("failed to copy spec: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Update config with temp file path
	config.SpecPath = tmpFile.Name()

	// Generate
	gen, err := NewGenerator(config)
	if err != nil {
		return err
	}

	if err := gen.LoadSpec(); err != nil {
		return err
	}

	return gen.Generate()
}
