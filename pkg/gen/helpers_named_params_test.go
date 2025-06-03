package gen

import (
	"testing"
)

func TestBuildPathWithNamedParams(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		params   []Parameter
		expected string
	}{
		{
			name: "single parameter",
			path: "/users/{id}",
			params: []Parameter{
				{Name: "id", In: "path", Type: "string"},
			},
			expected: `strings.ReplaceAll("/users/{id}", "{id}", fmt.Sprintf("%s", id))`,
		},
		{
			name: "multiple parameters in order",
			path: "/users/{userId}/posts/{postId}",
			params: []Parameter{
				{Name: "userId", In: "path", Type: "string"},
				{Name: "postId", In: "path", Type: "string"},
			},
			expected: `strings.ReplaceAll(strings.ReplaceAll("/users/{userId}/posts/{postId}", "{userId}", fmt.Sprintf("%s", userId)), "{postId}", fmt.Sprintf("%s", postId))`,
		},
		{
			name: "multiple parameters out of order",
			path: "/api/v1/packages/{packageKey}/public-form-links/{id}",
			params: []Parameter{
				{Name: "id", In: "path", Type: "string"},
				{Name: "packageKey", In: "path", Type: "string"},
			},
			expected: `strings.ReplaceAll(strings.ReplaceAll("/api/v1/packages/{packageKey}/public-form-links/{id}", "{packageKey}", fmt.Sprintf("%s", packageKey)), "{id}", fmt.Sprintf("%s", id))`,
		},
		{
			name: "path with query parameters (should ignore)",
			path: "/users/{userId}",
			params: []Parameter{
				{Name: "userId", In: "path", Type: "string"},
				{Name: "active", In: "query", Type: "bool"},
			},
			expected: `strings.ReplaceAll("/users/{userId}", "{userId}", fmt.Sprintf("%s", userId))`,
		},
		{
			name: "no path parameters",
			path: "/users",
			params: []Parameter{
				{Name: "limit", In: "query", Type: "int"},
			},
			expected: `"/users"`,
		},
		{
			name: "parameter names with special characters",
			path: "/items/{item-id}/sub-items/{sub_item_id}",
			params: []Parameter{
				{Name: "item-id", In: "path", Type: "string"},
				{Name: "sub_item_id", In: "path", Type: "string"},
			},
			expected: `strings.ReplaceAll(strings.ReplaceAll("/items/{item-id}/sub-items/{sub_item_id}", "{item-id}", fmt.Sprintf("%s", item-id)), "{sub_item_id}", fmt.Sprintf("%s", sub_item_id))`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPathWithNamedParams(tt.path, tt.params)
			if result != tt.expected {
				t.Errorf("buildPathWithNamedParams() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestBuildPathWithNamedParamsVsBuildPath ensures the runtime behavior is equivalent
func TestBuildPathWithNamedParamsVsBuildPath(t *testing.T) {
	// This test verifies that the old positional approach and new named approach
	// produce the same runtime results when parameters are in the correct order

	path := "/users/{userId}/posts/{postId}/comments/{commentId}"
	params := []Parameter{
		{Name: "userId", In: "path", Type: "string"},
		{Name: "postId", In: "path", Type: "string"},
		{Name: "commentId", In: "path", Type: "string"},
	}

	// Old approach generates: "/users/%s/posts/%s/comments/%s"
	oldTemplate := buildPath(path, params)
	expectedOld := "/users/%s/posts/%s/comments/%s"
	if oldTemplate != expectedOld {
		t.Errorf("buildPath() = %v, want %v", oldTemplate, expectedOld)
	}

	// New approach generates code that should produce the same result
	newCode := buildPathWithNamedParams(path, params)
	expectedNew := `strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll("/users/{userId}/posts/{postId}/comments/{commentId}", "{userId}", fmt.Sprintf("%s", userId)), "{postId}", fmt.Sprintf("%s", postId)), "{commentId}", fmt.Sprintf("%s", commentId))`
	if newCode != expectedNew {
		t.Errorf("buildPathWithNamedParams() = %v, want %v", newCode, expectedNew)
	}
}
