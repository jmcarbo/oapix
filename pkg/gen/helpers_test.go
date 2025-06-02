package gen

import (
	"testing"
)

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello_world", "HelloWorld"},
		{"hello-world", "HelloWorld"},
		{"hello world", "HelloWorld"},
		{"helloWorld", "HelloWorld"},
		{"HelloWorld", "HelloWorld"},
		{"user_id", "UserID"},
		{"api_key", "APIKey"},
		{"http_request", "HTTPRequest"},
		{"json_data", "JSONData"},
		{"XML_parser", "XMLParser"},
		{"", ""},
		{"a", "A"},
		{"123test", "123Test"},
		{"test123", "Test123"},
		{"test_123_abc", "Test123Abc"},
		{"API_URL", "APIURL"},
		// Test cases with underscores followed by numbers
		{"get_user_2", "GetUser2"},
		{"create_item_v2", "CreateItemV2"},
		{"update_record_3_data", "UpdateRecord3Data"},
		{"delete_file_2023", "DeleteFile2023"},
		{"list_items_v2_1", "ListItemsV21"},
		{"process_batch_2_items", "ProcessBatch2Items"},
		{"get_2_users", "Get2Users"},
		{"fetch_v2_data", "FetchV2Data"},
		{"upload_file2", "UploadFile2"},
		{"download_file_2", "DownloadFile2"},
		{"method_2a", "Method2A"},
		{"api_v2_users", "APIV2Users"},
		{"get_v2", "GetV2"},
		{"item_2_name", "Item2Name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello_world", "helloWorld"},
		{"hello-world", "helloWorld"},
		{"hello world", "helloWorld"},
		{"HelloWorld", "helloWorld"},
		{"user_id", "userID"},
		{"API_key", "apiKey"},
		{"HTTPRequest", "httpRequest"},
		{"ID", "id"},
		{"", ""},
		{"a", "a"},
		{"A", "a"},
		{"ABC", "abc"},
		{"ABCDef", "abcDef"},
		{"XMLParser", "xmlParser"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toCamelCase(tt.input)
			if got != tt.want {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HelloWorld", "hello_world"},
		{"helloWorld", "hello_world"},
		{"hello_world", "hello_world"},
		{"HTTPRequest", "http_request"},
		{"userID", "user_id"},
		{"ID", "id"},
		{"APIKey", "api_key"},
		{"", ""},
		{"A", "a"},
		{"ABC", "abc"},
		{"TestHTTPServer", "test_http_server"},
		{"already_snake_case", "already_snake_case"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToKebabCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"HelloWorld", "hello-world"},
		{"helloWorld", "hello-world"},
		{"hello_world", "hello-world"},
		{"hello-world", "hello-world"},
		{"HTTPRequest", "http-request"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := toKebabCase(tt.input)
			if got != tt.want {
				t.Errorf("toKebabCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeGoName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"validName", "validName"},
		{"valid_name", "valid_name"},
		{"123invalid", "V123invalid"},
		{"invalid-name", "invalid_name"},
		{"invalid name", "invalid_name"},
		{"type", "type_"},
		{"package", "package_"},
		{"func", "func_"},
		{"return", "return_"},
		{"", ""},
		{"_validName", "_validName"},
		{"$invalid", "V_invalid"},
		{"@special#chars%", "V_special_chars_"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeGoName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeGoName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsBuiltinType(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"string", true},
		{"int", true},
		{"int64", true},
		{"float64", true},
		{"bool", true},
		{"interface{}", true},
		{"error", true},
		{"[]string", true},
		{"[]int", true},
		{"map[string]string", true},
		{"map[string]interface{}", true},
		{"User", false},
		{"CustomType", false},
		{"*string", false},
		{"time.Time", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isBuiltinType(tt.input)
			if got != tt.want {
				t.Errorf("isBuiltinType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNeedsPointer(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		want  bool
	}{
		{
			name: "required non-nullable",
			field: Field{
				Required: true,
				Nullable: false,
			},
			want: false,
		},
		{
			name: "required nullable",
			field: Field{
				Required: true,
				Nullable: true,
			},
			want: true,
		},
		{
			name: "optional non-nullable",
			field: Field{
				Required: false,
				Nullable: false,
			},
			want: true,
		},
		{
			name: "optional nullable",
			field: Field{
				Required: false,
				Nullable: true,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := needsPointer(tt.field)
			if got != tt.want {
				t.Errorf("needsPointer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"user", "users"},
		{"User", "Users"},
		{"person", "people"},
		{"Person", "People"},
		{"child", "children"},
		{"man", "men"},
		{"woman", "women"},
		{"city", "cities"},
		{"country", "countries"},
		{"boy", "boys"},
		{"box", "boxes"},
		{"class", "classes"},
		{"bus", "buses"},
		{"dish", "dishes"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := pluralize(tt.input)
			if got != tt.want {
				t.Errorf("pluralize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"users", "user"},
		{"Users", "User"},
		{"people", "person"},
		{"People", "Person"},
		{"children", "child"},
		{"men", "man"},
		{"women", "woman"},
		{"cities", "city"},
		{"countries", "country"},
		{"boys", "boy"},
		{"boxes", "box"},
		{"classes", "class"},
		{"buses", "bus"},
		{"dishes", "dish"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := singularize(tt.input)
			if got != tt.want {
				t.Errorf("singularize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"/users", []string{}},
		{"/users/{id}", []string{"id"}},
		{"/users/{userId}/posts/{postId}", []string{"userId", "postId"}},
		{"/api/v1/users/{id}/profile", []string{"id"}},
		{"/{version}/users/{id}", []string{"version", "id"}},
		{"", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := extractPathParams(tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("extractPathParams(%q) returned %d params, want %d", tt.path, len(got), len(tt.want))
				return
			}
			for i, param := range got {
				if param != tt.want[i] {
					t.Errorf("extractPathParams(%q)[%d] = %q, want %q", tt.path, i, param, tt.want[i])
				}
			}
		})
	}
}

func TestFilterParamsByIn(t *testing.T) {
	params := []Parameter{
		{Name: "id", In: "path"},
		{Name: "userId", In: "path"},
		{Name: "page", In: "query"},
		{Name: "limit", In: "query"},
		{Name: "X-API-Key", In: "header"},
		{Name: "session", In: "cookie"},
	}

	tests := []struct {
		in        string
		wantLen   int
		wantFirst string
	}{
		{"path", 2, "id"},
		{"query", 2, "page"},
		{"header", 1, "X-API-Key"},
		{"cookie", 1, "session"},
		{"body", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := filterParamsByIn(params, tt.in)
			if len(got) != tt.wantLen {
				t.Errorf("filterParamsByIn(%q) returned %d params, want %d", tt.in, len(got), tt.wantLen)
			}
			if tt.wantLen > 0 && got[0].Name != tt.wantFirst {
				t.Errorf("filterParamsByIn(%q)[0].Name = %q, want %q", tt.in, got[0].Name, tt.wantFirst)
			}
		})
	}
}

func TestBuildMethodSignature(t *testing.T) {
	tests := []struct {
		name string
		op   Operation
		want string
	}{
		{
			name: "simple GET",
			op: Operation{
				Name:   "GetUser",
				Method: "GET",
			},
			want: "ctx context.Context",
		},
		{
			name: "with path parameter",
			op: Operation{
				Name:   "GetUser",
				Method: "GET",
				Parameters: []Parameter{
					{Name: "id", In: "path", Type: "int64"},
				},
			},
			want: "ctx context.Context, id int64",
		},
		{
			name: "with request body",
			op: Operation{
				Name:   "CreateUser",
				Method: "POST",
				RequestBody: &RequestBody{
					Type: "CreateUserRequest",
				},
			},
			want: "ctx context.Context, req CreateUserRequest",
		},
		{
			name: "with query parameters",
			op: Operation{
				Name:   "ListUsers",
				Method: "GET",
				Parameters: []Parameter{
					{Name: "page", In: "query", Type: "int"},
					{Name: "limit", In: "query", Type: "int"},
				},
			},
			want: "ctx context.Context, params *ListUsersParams",
		},
		{
			name: "complex operation",
			op: Operation{
				Name:   "UpdateUserPost",
				Method: "PUT",
				Parameters: []Parameter{
					{Name: "userId", In: "path", Type: "int64"},
					{Name: "postId", In: "path", Type: "int64"},
					{Name: "X-Request-ID", In: "header", Type: "string"},
				},
				RequestBody: &RequestBody{
					Type: "UpdatePostRequest",
				},
			},
			want: "ctx context.Context, userId int64, postId int64, req UpdatePostRequest, params *UpdateUserPostParams",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildMethodSignature(tt.op)
			if got != tt.want {
				t.Errorf("buildMethodSignature() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGoDoc(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		prefix string
		want   string
	}{
		{
			name:   "single line",
			input:  "This is a comment",
			prefix: "",
			want:   "// This is a comment",
		},
		{
			name:   "multi line",
			input:  "Line 1\nLine 2\nLine 3",
			prefix: "",
			want:   "// Line 1\n// Line 2\n// Line 3",
		},
		{
			name:   "with prefix",
			input:  "Indented comment",
			prefix: "\t",
			want:   "\t// Indented comment",
		},
		{
			name:   "empty lines",
			input:  "Line 1\n\nLine 3",
			prefix: "",
			want:   "// Line 1\n//\n// Line 3",
		},
		{
			name:   "empty string",
			input:  "",
			prefix: "",
			want:   "",
		},
		{
			name:   "whitespace only",
			input:  "  \n  \n  ",
			prefix: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := goDoc(tt.input, tt.prefix)
			if got != tt.want {
				t.Errorf("goDoc(%q, %q) = %q, want %q", tt.input, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestIncDec(t *testing.T) {
	tests := []struct {
		fn    func(int) int
		input int
		want  int
	}{
		{inc, 5, 6},
		{inc, -1, 0},
		{inc, 0, 1},
		{dec, 5, 4},
		{dec, 0, -1},
		{dec, 1, 0},
	}

	for _, tt := range tests {
		got := tt.fn(tt.input)
		if got != tt.want {
			t.Errorf("function(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
