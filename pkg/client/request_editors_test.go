package client

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBearerTokenAuth(t *testing.T) {
	editor := BearerTokenAuth("test-token")
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("BearerTokenAuth failed: %v", err)
	}

	auth := req.Header.Get("Authorization")
	expected := "Bearer test-token"
	if auth != expected {
		t.Errorf("Expected Authorization header %q, got %q", expected, auth)
	}
}

func TestBasicAuth(t *testing.T) {
	editor := BasicAuth("user", "pass")
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("BasicAuth failed: %v", err)
	}

	auth := req.Header.Get("Authorization")
	expectedCreds := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	expected := "Basic " + expectedCreds
	if auth != expected {
		t.Errorf("Expected Authorization header %q, got %q", expected, auth)
	}
}

func TestAPIKeyAuth(t *testing.T) {
	editor := APIKeyAuth("X-API-Key", "my-api-key")
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("APIKeyAuth failed: %v", err)
	}

	apiKey := req.Header.Get("X-API-Key")
	if apiKey != "my-api-key" {
		t.Errorf("Expected X-API-Key header %q, got %q", "my-api-key", apiKey)
	}
}

func TestQueryParamAuth(t *testing.T) {
	editor := QueryParamAuth("api_key", "secret123")
	req := httptest.NewRequest("GET", "http://example.com/path", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("QueryParamAuth failed: %v", err)
	}

	apiKey := req.URL.Query().Get("api_key")
	if apiKey != "secret123" {
		t.Errorf("Expected api_key query param %q, got %q", "secret123", apiKey)
	}
}

func TestCustomHeaders(t *testing.T) {
	headers := map[string]string{
		"X-Custom-1": "value1",
		"X-Custom-2": "value2",
	}
	editor := CustomHeaders(headers)
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("CustomHeaders failed: %v", err)
	}

	for k, v := range headers {
		got := req.Header.Get(k)
		if got != v {
			t.Errorf("Expected header %s=%q, got %q", k, v, got)
		}
	}
}

func TestChainRequestEditors(t *testing.T) {
	var calls []string

	editor1 := func(ctx context.Context, req *http.Request) error {
		calls = append(calls, "editor1")
		req.Header.Set("X-Editor-1", "true")
		return nil
	}

	editor2 := func(ctx context.Context, req *http.Request) error {
		calls = append(calls, "editor2")
		req.Header.Set("X-Editor-2", "true")
		return nil
	}

	chained := ChainRequestEditors(editor1, editor2)
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := chained(context.Background(), req)
	if err != nil {
		t.Fatalf("ChainRequestEditors failed: %v", err)
	}

	// Check execution order
	if len(calls) != 2 || calls[0] != "editor1" || calls[1] != "editor2" {
		t.Errorf("Expected calls [editor1, editor2], got %v", calls)
	}

	// Check headers
	if req.Header.Get("X-Editor-1") != "true" {
		t.Error("X-Editor-1 header not set")
	}
	if req.Header.Get("X-Editor-2") != "true" {
		t.Error("X-Editor-2 header not set")
	}
}

func TestConditionalRequestEditor(t *testing.T) {
	tests := []struct {
		name      string
		condition func(context.Context, *http.Request) bool
		shouldRun bool
	}{
		{
			name: "condition true",
			condition: func(ctx context.Context, req *http.Request) bool {
				return true
			},
			shouldRun: true,
		},
		{
			name: "condition false",
			condition: func(ctx context.Context, req *http.Request) bool {
				return false
			},
			shouldRun: false,
		},
		{
			name: "condition based on method",
			condition: func(ctx context.Context, req *http.Request) bool {
				return req.Method == "POST"
			},
			shouldRun: false, // We're using GET
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editor := ConditionalRequestEditor(tt.condition, CustomHeader("X-Conditional", "true"))
			req := httptest.NewRequest("GET", "http://example.com", nil)

			err := editor(context.Background(), req)
			if err != nil {
				t.Fatalf("ConditionalRequestEditor failed: %v", err)
			}

			header := req.Header.Get("X-Conditional")
			if tt.shouldRun && header != "true" {
				t.Error("Expected header to be set, but it wasn't")
			}
			if !tt.shouldRun && header == "true" {
				t.Error("Expected header not to be set, but it was")
			}
		})
	}
}

// Mock OAuth2 token source for testing
type mockTokenSource struct {
	token string
	err   error
}

func (m *mockTokenSource) Token(ctx context.Context) (string, error) {
	return m.token, m.err
}

func TestOAuth2Auth(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		err       error
		wantError bool
	}{
		{
			name:  "successful token",
			token: "oauth-token-123",
		},
		{
			name:      "token error",
			err:       context.DeadlineExceeded,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenSource := &mockTokenSource{token: tt.token, err: tt.err}
			editor := OAuth2Auth(tokenSource)
			req := httptest.NewRequest("GET", "http://example.com", nil)

			err := editor(context.Background(), req)
			if (err != nil) != tt.wantError {
				t.Errorf("OAuth2Auth error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError {
				auth := req.Header.Get("Authorization")
				expected := "Bearer " + tt.token
				if auth != expected {
					t.Errorf("Expected Authorization header %q, got %q", expected, auth)
				}
			}
		})
	}
}

func TestUserAgent(t *testing.T) {
	editor := UserAgent("test-client/1.0")
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("UserAgent failed: %v", err)
	}

	ua := req.Header.Get("User-Agent")
	if ua != "test-client/1.0" {
		t.Errorf("Expected User-Agent %q, got %q", "test-client/1.0", ua)
	}
}

func TestRequestID(t *testing.T) {
	generator := func() string { return "req-123" }
	editor := RequestID("X-Request-ID", generator)
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("RequestID failed: %v", err)
	}

	reqID := req.Header.Get("X-Request-ID")
	if reqID != "req-123" {
		t.Errorf("Expected X-Request-ID %q, got %q", "req-123", reqID)
	}
}

func TestCookieAuth(t *testing.T) {
	cookie1 := &http.Cookie{Name: "session", Value: "abc123"}
	cookie2 := &http.Cookie{Name: "user", Value: "john"}

	editor := CookieAuth(cookie1, cookie2)
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("CookieAuth failed: %v", err)
	}

	cookies := req.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("Expected 2 cookies, got %d", len(cookies))
	}

	// Check cookies
	cookieMap := make(map[string]string)
	for _, c := range cookies {
		cookieMap[c.Name] = c.Value
	}

	if cookieMap["session"] != "abc123" {
		t.Errorf("Expected session cookie value %q, got %q", "abc123", cookieMap["session"])
	}
	if cookieMap["user"] != "john" {
		t.Errorf("Expected user cookie value %q, got %q", "john", cookieMap["user"])
	}
}

func TestContentType(t *testing.T) {
	editor := ContentType("application/json; charset=utf-8")
	req := httptest.NewRequest("POST", "http://example.com", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("ContentType failed: %v", err)
	}

	ct := req.Header.Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type %q, got %q", "application/json; charset=utf-8", ct)
	}
}

func TestAcceptHeader(t *testing.T) {
	editor := AcceptHeader("application/xml")
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("AcceptHeader failed: %v", err)
	}

	accept := req.Header.Get("Accept")
	if accept != "application/xml" {
		t.Errorf("Expected Accept header %q, got %q", "application/xml", accept)
	}
}

func TestTimeout(t *testing.T) {
	editor := Timeout(5 * time.Second)
	req := httptest.NewRequest("GET", "http://example.com", nil)

	err := editor(context.Background(), req)
	if err != nil {
		t.Fatalf("Timeout failed: %v", err)
	}

	// Check that context has deadline
	deadline, ok := req.Context().Deadline()
	if !ok {
		t.Error("Expected context to have deadline")
	}

	// Check deadline is approximately 5 seconds from now
	remaining := time.Until(deadline)
	if remaining < 4*time.Second || remaining > 6*time.Second {
		t.Errorf("Expected deadline ~5s from now, got %v", remaining)
	}
}

// Integration test with BaseClient
func TestRequestEditorsWithBaseClient(t *testing.T) {
	// Create a test server that echoes headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Echo some headers back
		response := map[string]string{
			"authorization": r.Header.Get("Authorization"),
			"user-agent":    r.Header.Get("User-Agent"),
			"x-api-key":     r.Header.Get("X-API-Key"),
			"x-custom":      r.Header.Get("X-Custom"),
		}

		// Check query params
		if apiKey := r.URL.Query().Get("api_key"); apiKey != "" {
			response["query_api_key"] = apiKey
		}

		// Simple JSON response
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	// Create client with multiple request editors
	config := &Config{
		BaseURL: server.URL,
		RequestEditors: []RequestEditor{
			BearerTokenAuth("test-token"),
			UserAgent("test-client/1.0"),
			CustomHeader("X-Custom", "custom-value"),
		},
	}

	client, err := NewBaseClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Add another editor after creation
	client.AddRequestEditor(APIKeyAuth("X-API-Key", "api-key-123"))

	// Make request
	ctx := context.Background()
	resp, err := client.Request(ctx, "GET", "/test", nil)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequestEditorError(t *testing.T) {
	// Create an editor that returns an error
	failingEditor := func(ctx context.Context, req *http.Request) error {
		return context.DeadlineExceeded
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &Config{
		BaseURL: server.URL,
		RequestEditors: []RequestEditor{
			CustomHeader("X-Test", "value"),
			failingEditor,
			CustomHeader("X-Never-Set", "value"), // Should not be reached
		},
	}

	client, err := NewBaseClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	_, err = client.Request(ctx, "GET", "/test", nil)
	if err == nil {
		t.Error("Expected error from failing editor, got nil")
	}

	expectedError := "request editor failed"
	if err != nil && !contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing %q, got %q", expectedError, err.Error())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}
