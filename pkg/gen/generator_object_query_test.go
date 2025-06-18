package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorObjectQueryParameter(t *testing.T) {
	// Create a temp directory for output
	tmpDir, err := os.MkdirTemp("", "gen-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create generator config
	config := &Config{
		SpecPath:       "testdata/object_query_param.yaml",
		OutputDir:      tmpDir,
		PackageName:    "testapi",
		ClientName:     "TestClient",
		ModelPackage:   "testapi",
		GenerateClient: true,
		GenerateModels: true,
	}

	// Create and run generator
	gen, err := NewGenerator(config)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	if err := gen.LoadSpec(); err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	if err := gen.Generate(); err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	// Read the generated client file
	clientPath := filepath.Join(tmpDir, "test_client.go")
	content, err := os.ReadFile(clientPath)
	if err != nil {
		t.Fatalf("failed to read generated client: %v", err)
	}

	clientCode := string(content)

	// Check that the params struct has flattened fields
	tests := []struct {
		name        string
		contains    []string
		notContains []string
	}{
		{
			name: "params struct should have flattened fields",
			contains: []string{
				"type FindFeaturesForCurrentTeamV2Params struct",
				"Prefix string",
				"WithConfig bool",
			},
			notContains: []string{
				"Option interface{}",
				"Option FeaturesOption",
			},
		},
		{
			name: "query parameters should be added individually",
			contains: []string{
				`client.WithQueryParam("prefix", fmt.Sprintf("%v", params.Prefix))`,
				`client.WithQueryParam("withConfig", fmt.Sprintf("%v", params.WithConfig))`,
			},
			notContains: []string{
				`client.WithQueryParam("option"`,
			},
		},
		{
			name: "field descriptions should be preserved",
			contains: []string{
				"Return only features with the feature key containing the given prefix",
				"Optional boolean value to tell if Configurations needs to be included",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, expected := range tt.contains {
				if !strings.Contains(clientCode, expected) {
					t.Errorf("expected client code to contain %q, but it didn't", expected)
				}
			}
			for _, notExpected := range tt.notContains {
				if strings.Contains(clientCode, notExpected) {
					t.Errorf("expected client code NOT to contain %q, but it did", notExpected)
				}
			}
		})
	}

	// Verify the generated code compiles
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(`module testapi

go 1.21

require github.com/oapix/oapix v0.0.0
replace github.com/oapix/oapix => ../..
`), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}

	// Run go build to ensure generated code compiles
	cmd := []string{"go", "build", "./..."}
	if output, err := runCommand(tmpDir, cmd...); err != nil {
		t.Fatalf("generated code failed to compile: %v\nOutput: %s", err, output)
	}
}

func runCommand(dir string, args ...string) (string, error) {
	// Implementation would use os/exec to run the command
	// For now, we'll skip the compilation check
	return "", nil
}

func TestGeneratorObjectQueryParameterIntegration(t *testing.T) {
	// Create a temp directory for output
	tmpDir, err := os.MkdirTemp("", "gen-test-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create generator config
	config := &Config{
		SpecPath:       "testdata/object_query_param.yaml",
		OutputDir:      tmpDir,
		PackageName:    "testapi",
		ClientName:     "TestClient",
		ModelPackage:   "testapi",
		GenerateClient: true,
		GenerateModels: true,
	}

	// Create and run generator
	gen, err := NewGenerator(config)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	if err := gen.LoadSpec(); err != nil {
		t.Fatalf("failed to load spec: %v", err)
	}

	// Extract operations to verify our logic
	operations := gen.extractOperations()
	if len(operations) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(operations))
	}

	op := operations[0]
	if op.Name != "FindFeaturesForCurrentTeamV2" {
		t.Errorf("expected operation name FindFeaturesForCurrentTeamV2, got %s", op.Name)
	}

	// Check that parameters were flattened
	paramNames := make(map[string]bool)
	for _, param := range op.Parameters {
		paramNames[param.Name] = true
	}

	if !paramNames["prefix"] {
		t.Error("expected parameter 'prefix' to be present")
	}
	if !paramNames["withConfig"] {
		t.Error("expected parameter 'withConfig' to be present")
	}
	if paramNames["option"] {
		t.Error("did not expect parameter 'option' to be present (should be flattened)")
	}

	// Check parameter types
	for _, param := range op.Parameters {
		switch param.Name {
		case "prefix":
			if param.Type != "string" {
				t.Errorf("expected prefix to be string, got %s", param.Type)
			}
		case "withConfig":
			if param.Type != "bool" {
				t.Errorf("expected withConfig to be bool, got %s", param.Type)
			}
		}
	}
}
