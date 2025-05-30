package gen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				PackageName: "testpkg",
				OutputDir:   "./output",
			},
			wantErr: false,
		},
		{
			name: "missing package name",
			config: &Config{
				OutputDir: "./output",
			},
			wantErr: true,
		},
		{
			name: "with custom client name",
			config: &Config{
				PackageName: "testpkg",
				ClientName:  "MyAPIClient",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGenerator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && gen == nil {
				t.Error("NewGenerator() returned nil generator")
			}
		})
	}
}

func TestLoadSpec(t *testing.T) {
	// Create a temporary spec file
	specContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
        email:
          type: string
`

	tmpFile, err := os.CreateTemp("", "test-spec-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.WriteString(specContent); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	config := &Config{
		SpecPath:    tmpFile.Name(),
		PackageName: "testapi",
	}

	gen, err := NewGenerator(config)
	if err != nil {
		t.Fatal(err)
	}

	if err := gen.LoadSpec(); err != nil {
		t.Errorf("LoadSpec() error = %v", err)
	}

	if gen.spec == nil {
		t.Error("LoadSpec() did not load specification")
	}
}

func TestSchemaToGoType(t *testing.T) {
	gen := &Generator{}

	stringType := openapi3.Types{"string"}
	integerType := openapi3.Types{"integer"}
	numberType := openapi3.Types{"number"}
	booleanType := openapi3.Types{"boolean"}
	arrayType := openapi3.Types{"array"}
	objectType := openapi3.Types{"object"}

	tests := []struct {
		name   string
		schema *openapi3.Schema
		want   string
	}{
		{
			name:   "string type",
			schema: &openapi3.Schema{Type: &stringType},
			want:   "string",
		},
		{
			name:   "integer type",
			schema: &openapi3.Schema{Type: &integerType},
			want:   "int64",
		},
		{
			name:   "int32 format",
			schema: &openapi3.Schema{Type: &integerType, Format: "int32"},
			want:   "int32",
		},
		{
			name:   "number type",
			schema: &openapi3.Schema{Type: &numberType},
			want:   "float64",
		},
		{
			name:   "float format",
			schema: &openapi3.Schema{Type: &numberType, Format: "float"},
			want:   "float32",
		},
		{
			name:   "boolean type",
			schema: &openapi3.Schema{Type: &booleanType},
			want:   "bool",
		},
		{
			name:   "date-time format",
			schema: &openapi3.Schema{Type: &stringType, Format: "date-time"},
			want:   "time.Time",
		},
		{
			name: "array of strings",
			schema: &openapi3.Schema{
				Type: &arrayType,
				Items: &openapi3.SchemaRef{
					Value: &openapi3.Schema{Type: &stringType},
				},
			},
			want: "[]string",
		},
		{
			name: "object with additional properties",
			schema: &openapi3.Schema{
				Type: &objectType,
				AdditionalProperties: openapi3.AdditionalProperties{
					Has: &[]bool{true}[0],
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{Type: &stringType},
					},
				},
			},
			want: "map[string]string",
		},
		{
			name:   "nil schema",
			schema: nil,
			want:   "interface{}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gen.schemaToGoType(tt.schema)
			if got != tt.want {
				t.Errorf("schemaToGoType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractModels(t *testing.T) {
	objectType := openapi3.Types{"object"}
	stringType := openapi3.Types{"string"}
	integerType := openapi3.Types{"integer"}
	
	spec := &openapi3.T{
		Components: &openapi3.Components{
			Schemas: map[string]*openapi3.SchemaRef{
				"User": {
					Value: &openapi3.Schema{
						Type: &objectType,
						Properties: map[string]*openapi3.SchemaRef{
							"id": {
								Value: &openapi3.Schema{Type: &integerType, Format: "int64"},
							},
							"name": {
								Value: &openapi3.Schema{Type: &stringType},
							},
							"email": {
								Value: &openapi3.Schema{Type: &stringType},
							},
						},
						Required: []string{"id", "name"},
					},
				},
				"Status": {
					Value: &openapi3.Schema{
						Type: &stringType,
						Enum: []interface{}{"active", "inactive", "pending"},
					},
				},
			},
		},
	}

	gen := &Generator{spec: spec}
	models := gen.extractModels()

	if len(models) != 2 {
		t.Errorf("extractModels() returned %d models, want 2", len(models))
	}

	// Check User model
	var userModel *Model
	for i := range models {
		if models[i].Name == "User" {
			userModel = &models[i]
			break
		}
	}

	if userModel == nil {
		t.Fatal("User model not found")
	}

	if len(userModel.Fields) != 3 {
		t.Errorf("User model has %d fields, want 3", len(userModel.Fields))
	}

	// Check required fields
	var idField, nameField *Field
	for i := range userModel.Fields {
		switch userModel.Fields[i].Name {
		case "ID":
			idField = &userModel.Fields[i]
		case "Name":
			nameField = &userModel.Fields[i]
		}
	}

	if idField == nil || !idField.Required {
		t.Error("ID field should be required")
	}
	if nameField == nil || !nameField.Required {
		t.Error("Name field should be required")
	}

	// Check Status enum
	var statusModel *Model
	for i := range models {
		if models[i].Name == "Status" {
			statusModel = &models[i]
			break
		}
	}

	if statusModel == nil {
		t.Fatal("Status model not found")
	}

	if !statusModel.IsEnum {
		t.Error("Status model should be an enum")
	}

	if len(statusModel.EnumValues) != 3 {
		t.Errorf("Status enum has %d values, want 3", len(statusModel.EnumValues))
	}
}

func TestGenerateFromReader(t *testing.T) {
	specContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: Success
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`

	tmpDir, err := os.MkdirTemp("", "gen-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	config := &Config{
		OutputDir:      tmpDir,
		PackageName:    "testapi",
		GenerateModels: true,
		GenerateClient: true,
	}

	reader := strings.NewReader(specContent)
	if err := GenerateFromReader(reader, config); err != nil {
		t.Fatalf("GenerateFromReader() error = %v", err)
	}

	// Check that files were generated
	modelsPath := filepath.Join(tmpDir, "models.go")
	if _, err := os.Stat(modelsPath); os.IsNotExist(err) {
		t.Error("models.go was not generated")
	}

	clientPath := filepath.Join(tmpDir, "client.go")
	if _, err := os.Stat(clientPath); os.IsNotExist(err) {
		t.Error("client.go was not generated")
	}
}

func TestGenerateOperationName(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   string
	}{
		{"GET", "/users", "GetUsers"},
		{"POST", "/users", "PostUsers"},
		{"GET", "/users/{id}", "GetUsers"},
		{"PUT", "/users/{id}", "PutUsers"},
		{"DELETE", "/users/{id}", "DeleteUsers"},
		{"GET", "/users/{id}/posts", "GetUsersPosts"},
		{"POST", "/api/v1/users", "PostAPIV1Users"},
		{"GET", "/", "Get"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			got := generateOperationName(tt.method, tt.path)
			if got != tt.want {
				t.Errorf("generateOperationName(%q, %q) = %q, want %q", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestBuildPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		params []Parameter
		want   string
	}{
		{
			name: "no parameters",
			path: "/users",
			want: "/users",
		},
		{
			name: "single path parameter",
			path: "/users/{id}",
			params: []Parameter{
				{Name: "id", In: "path"},
			},
			want: "/users/%s",
		},
		{
			name: "multiple path parameters",
			path: "/users/{userId}/posts/{postId}",
			params: []Parameter{
				{Name: "userId", In: "path"},
				{Name: "postId", In: "path"},
			},
			want: "/users/%s/posts/%s",
		},
		{
			name: "mixed parameters",
			path: "/users/{id}",
			params: []Parameter{
				{Name: "id", In: "path"},
				{Name: "filter", In: "query"},
			},
			want: "/users/%s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPath(tt.path, tt.params)
			if got != tt.want {
				t.Errorf("buildPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateLoading(t *testing.T) {
	// Test embedded templates
	config := &Config{
		PackageName: "test",
	}

	gen, err := NewGenerator(config)
	if err != nil {
		t.Fatal(err)
	}

	if gen.templates == nil {
		t.Error("templates should be loaded")
	}

	// Check that required templates exist
	requiredTemplates := []string{"models", "client"}
	for _, name := range requiredTemplates {
		tmpl := gen.templates.Lookup(name)
		if tmpl == nil {
			t.Errorf("template %q not found", name)
		}
	}
}

func TestGenerateWithCompleteSpec(t *testing.T) {
	// Create a more complete spec for testing
	specContent := `
openapi: 3.0.0
info:
  title: Complete Test API
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
paths:
  /users:
    get:
      operationId: listUsers
      summary: List all users
      parameters:
        - name: page
          in: query
          schema:
            type: integer
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
    post:
      operationId: createUser
      summary: Create a new user
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateUserRequest'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
  /users/{id}:
    get:
      operationId: getUser
      summary: Get a user by ID
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
            format: int64
        - name: X-Request-ID
          in: header
          schema:
            type: string
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        '404':
          description: Not found
    put:
      operationId: updateUser
      summary: Update a user
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
            format: int64
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateUserRequest'
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
    delete:
      operationId: deleteUser
      summary: Delete a user
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
            format: int64
      responses:
        '204':
          description: No content
components:
  schemas:
    User:
      type: object
      required:
        - id
        - username
        - createdAt
      properties:
        id:
          type: integer
          format: int64
          description: Unique identifier
        username:
          type: string
          description: Username
        email:
          type: string
          format: email
          description: Email address
        fullName:
          type: string
          description: Full name
        status:
          $ref: '#/components/schemas/UserStatus'
        createdAt:
          type: string
          format: date-time
          description: Creation timestamp
        updatedAt:
          type: string
          format: date-time
          description: Last update timestamp
        tags:
          type: array
          items:
            type: string
        metadata:
          type: object
          additionalProperties:
            type: string
    UserStatus:
      type: string
      enum:
        - active
        - inactive
        - pending
      description: User status
    CreateUserRequest:
      type: object
      required:
        - username
        - email
      properties:
        username:
          type: string
        email:
          type: string
          format: email
        fullName:
          type: string
    UpdateUserRequest:
      type: object
      properties:
        email:
          type: string
          format: email
        fullName:
          type: string
        status:
          $ref: '#/components/schemas/UserStatus'
`

	tmpFile, err := os.CreateTemp("", "complete-spec-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.WriteString(specContent); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	tmpDir, err := os.MkdirTemp("", "gen-complete-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	config := &Config{
		SpecPath:       tmpFile.Name(),
		OutputDir:      tmpDir,
		PackageName:    "userapi",
		ClientName:     "UserAPIClient",
		GenerateModels: true,
		GenerateClient: true,
		Verbose:        true,
	}

	gen, err := NewGenerator(config)
	if err != nil {
		t.Fatal(err)
	}

	if err := gen.LoadSpec(); err != nil {
		t.Fatal(err)
	}

	if err := gen.Generate(); err != nil {
		t.Fatal(err)
	}

	// Verify generated files exist and can be read
	modelsContent, err := os.ReadFile(filepath.Join(tmpDir, "models.go"))
	if err != nil {
		t.Fatal(err)
	}

	// Check models content
	modelsStr := string(modelsContent)
	
	// Debug: print the generated content
	if testing.Verbose() {
		t.Logf("Generated models.go:\n%s", modelsStr)
	}
	
	// Use regexp to handle variable whitespace in formatted Go code
	expectedPatterns := []struct {
		pattern string
		desc    string
	}{
		{`type\s+User\s+struct`, "type User struct"},
		{`type\s+UserStatus\s+string`, "type UserStatus string"},
		{`type\s+CreateUserRequest\s+struct`, "type CreateUserRequest struct"},
		{`type\s+UpdateUserRequest\s+struct`, "type UpdateUserRequest struct"},
		{`ID\s+int64`, "ID int64"},
		{`Username\s+string`, "Username string"},
		{`Email\s+\*string`, "Email *string"},
		{`CreatedAt\s+time\.Time`, "CreatedAt time.Time"},
		{`Tags\s+\[\]string`, "Tags []string"},
		{`Metadata\s+map\[string\]string`, "Metadata map[string]string"},
	}

	for _, exp := range expectedPatterns {
		matched, err := regexp.MatchString(exp.pattern, modelsStr)
		if err != nil {
			t.Fatalf("Invalid regex pattern %q: %v", exp.pattern, err)
		}
		if !matched {
			t.Errorf("models.go should contain %q", exp.desc)
		}
	}

	clientContent, err := os.ReadFile(filepath.Join(tmpDir, "user_api_client.go"))
	if err != nil {
		t.Fatal(err)
	}

	// Check client content
	clientStr := string(clientContent)
	expectedClient := []string{
		"type UserAPIClient struct",
		"func NewUserAPIClient",
		"func (c *UserAPIClient) ListUsers",
		"func (c *UserAPIClient) CreateUser",
		"func (c *UserAPIClient) GetUser",
		"func (c *UserAPIClient) UpdateUser",
		"func (c *UserAPIClient) DeleteUser",
		"ListUsersParams struct",
		"GetUserParams struct",
		"Page int64",
		"Limit int64",
		"XRequestID string",
	}

	for _, expected := range expectedClient {
		if !strings.Contains(clientStr, expected) {
			t.Errorf("client.go should contain %q", expected)
		}
	}
}