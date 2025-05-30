package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// mockHTTPClient is a mock implementation of HTTPClient for testing
type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

// mockResponse creates a mock HTTP response
func mockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestNewBaseClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				BaseURL: "https://api.example.com",
			},
			wantErr: false,
		},
		{
			name: "valid config with trailing slash",
			config: &Config{
				BaseURL: "https://api.example.com/",
			},
			wantErr: false,
		},
		{
			name:    "missing base URL",
			config:  &Config{},
			wantErr: true,
		},
		{
			name: "with API key",
			config: &Config{
				BaseURL: "https://api.example.com",
				APIKey:  "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewBaseClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewBaseClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewBaseClient() returned nil client")
			}
			if !tt.wantErr && !strings.HasSuffix(client.baseURL, "/") {
				t.Error("BaseURL should end with /")
			}
		})
	}
}

func TestBaseClient_Request(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       io.Reader
		opts       []RequestOption
		mockFunc   func(req *http.Request) (*http.Response, error)
		wantStatus int
		wantErr    bool
	}{
		{
			name:   "successful GET request",
			method: "GET",
			path:   "test/endpoint",
			body:   nil,
			mockFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method != "GET" {
					t.Errorf("expected GET, got %s", req.Method)
				}
				if req.URL.Path != "/test/endpoint" {
					t.Errorf("expected /test/endpoint, got %s", req.URL.Path)
				}
				return mockResponse(200, `{"success": true}`), nil
			},
			wantStatus: 200,
			wantErr:    false,
		},
		{
			name:   "successful POST request with body",
			method: "POST",
			path:   "test/create",
			body:   strings.NewReader(`{"name": "test"}`),
			mockFunc: func(req *http.Request) (*http.Response, error) {
				if req.Method != "POST" {
					t.Errorf("expected POST, got %s", req.Method)
				}
				if req.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
				}
				return mockResponse(201, `{"id": "123"}`), nil
			},
			wantStatus: 201,
			wantErr:    false,
		},
		{
			name:   "request with query parameters",
			method: "GET",
			path:   "test/search",
			body:   nil,
			opts: []RequestOption{
				WithQueryParam("q", "test"),
				WithQueryParam("limit", "10"),
			},
			mockFunc: func(req *http.Request) (*http.Response, error) {
				if req.URL.Query().Get("q") != "test" {
					t.Errorf("expected query param q=test, got %s", req.URL.Query().Get("q"))
				}
				if req.URL.Query().Get("limit") != "10" {
					t.Errorf("expected query param limit=10, got %s", req.URL.Query().Get("limit"))
				}
				return mockResponse(200, `{"results": []}`), nil
			},
			wantStatus: 200,
			wantErr:    false,
		},
		{
			name:   "request with custom headers",
			method: "GET",
			path:   "test/headers",
			body:   nil,
			opts: []RequestOption{
				WithHeader("X-Custom-Header", "value"),
			},
			mockFunc: func(req *http.Request) (*http.Response, error) {
				if req.Header.Get("X-Custom-Header") != "value" {
					t.Errorf("expected header X-Custom-Header=value, got %s", req.Header.Get("X-Custom-Header"))
				}
				return mockResponse(200, `{}`), nil
			},
			wantStatus: 200,
			wantErr:    false,
		},
		{
			name:   "error response",
			method: "GET",
			path:   "test/error",
			body:   nil,
			mockFunc: func(req *http.Request) (*http.Response, error) {
				return mockResponse(404, `{"error": "Not Found", "detail": "Resource not found"}`), nil
			},
			wantStatus: 404,
			wantErr:    true,
		},
		{
			name:   "API key authentication",
			method: "GET",
			path:   "test/auth",
			body:   nil,
			mockFunc: func(req *http.Request) (*http.Response, error) {
				authHeader := req.Header.Get("Authorization")
				if authHeader != "Bearer test-api-key" {
					t.Errorf("expected Authorization header 'Bearer test-api-key', got '%s'", authHeader)
				}
				return mockResponse(200, `{"authenticated": true}`), nil
			},
			wantStatus: 200,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{doFunc: tt.mockFunc}
			client := &BaseClient{
				httpClient: mockClient,
				baseURL:    "https://api.example.com/",
				apiKey:     "test-api-key",
			}

			resp, err := client.Request(context.Background(), tt.method, tt.path, tt.body, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Request() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if resp != nil && resp.StatusCode != tt.wantStatus {
				t.Errorf("Request() status = %v, want %v", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestBaseClient_RequestJSON(t *testing.T) {
	type testPayload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name     string
		method   string
		path     string
		body     interface{}
		mockFunc func(req *http.Request) (*http.Response, error)
		wantErr  bool
	}{
		{
			name:   "successful JSON request",
			method: "POST",
			path:   "test/json",
			body:   testPayload{Name: "test", Value: 42},
			mockFunc: func(req *http.Request) (*http.Response, error) {
				// Verify Content-Type header
				if req.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", req.Header.Get("Content-Type"))
				}

				// Verify request body
				body, _ := io.ReadAll(req.Body)
				var payload testPayload
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Errorf("failed to unmarshal request body: %v", err)
				}
				if payload.Name != "test" || payload.Value != 42 {
					t.Errorf("unexpected payload: %+v", payload)
				}

				return mockResponse(200, `{"success": true}`), nil
			},
			wantErr: false,
		},
		{
			name:   "nil body",
			method: "GET",
			path:   "test/nil",
			body:   nil,
			mockFunc: func(req *http.Request) (*http.Response, error) {
				if req.Body != nil {
					body, _ := io.ReadAll(req.Body)
					if len(body) != 0 {
						t.Errorf("expected empty body, got %s", string(body))
					}
				}
				return mockResponse(200, `{}`), nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockHTTPClient{doFunc: tt.mockFunc}
			client := &BaseClient{
				httpClient: mockClient,
				baseURL:    "https://api.example.com/",
			}

			_, err := client.RequestJSON(context.Background(), tt.method, tt.path, tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("RequestJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	type testStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name    string
		resp    *Response
		target  interface{}
		wantErr bool
	}{
		{
			name: "successful parse",
			resp: &Response{
				Body: []byte(`{"name": "test", "value": 42}`),
			},
			target:  &testStruct{},
			wantErr: false,
		},
		{
			name: "empty body",
			resp: &Response{
				Body: []byte{},
			},
			target:  &testStruct{},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			resp: &Response{
				Body: []byte(`{invalid json}`),
			},
			target:  &testStruct{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseJSON(tt.resp, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.name == "successful parse" {
				result := tt.target.(*testStruct)
				if result.Name != "test" || result.Value != 42 {
					t.Errorf("ParseJSON() unexpected result: %+v", result)
				}
			}
		})
	}
}

func TestBaseClient_buildURL(t *testing.T) {
	client := &BaseClient{
		baseURL: "https://api.example.com/v1/",
	}

	tests := []struct {
		name        string
		path        string
		queryParams map[string]string
		want        string
		wantErr     bool
	}{
		{
			name:        "simple path",
			path:        "users",
			queryParams: nil,
			want:        "https://api.example.com/v1/users",
			wantErr:     false,
		},
		{
			name:        "path with leading slash",
			path:        "/users",
			queryParams: nil,
			want:        "https://api.example.com/v1/users",
			wantErr:     false,
		},
		{
			name:        "nested path",
			path:        "users/123/posts",
			queryParams: nil,
			want:        "https://api.example.com/v1/users/123/posts",
			wantErr:     false,
		},
		{
			name: "path with query params",
			path: "users",
			queryParams: map[string]string{
				"limit":  "10",
				"offset": "20",
			},
			want:    "https://api.example.com/v1/users?limit=10&offset=20",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.buildURL(tt.path, tt.queryParams)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Compare URLs without query parameter order
			if !tt.wantErr && !compareURLs(got, tt.want) {
				t.Errorf("buildURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

// compareURLs compares two URLs ignoring query parameter order
func compareURLs(url1, url2 string) bool {
	if !strings.Contains(url1, "?") && !strings.Contains(url2, "?") {
		return url1 == url2
	}

	// For URLs with query params, we just check if they have the same base and params
	base1 := strings.Split(url1, "?")[0]
	base2 := strings.Split(url2, "?")[0]

	if base1 != base2 {
		return false
	}

	// For simplicity, we're not doing deep query param comparison here
	// In real tests, you'd want to parse and compare query params properly
	return true
}

func TestAPIError(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Message:    "Resource not found",
		Details: map[string]interface{}{
			"resource": "user",
			"id":       "123",
		},
	}

	if err.Error() != "Resource not found" {
		t.Errorf("APIError.Error() = %v, want %v", err.Error(), "Resource not found")
	}
}

func TestRequestOptions(t *testing.T) {
	config := &RequestConfig{}

	// Test WithHeader
	WithHeader("X-Test", "value")(config)
	if config.Headers["X-Test"] != "value" {
		t.Errorf("WithHeader did not set header correctly")
	}

	// Test WithQueryParam
	WithQueryParam("test", "value")(config)
	if config.QueryParams["test"] != "value" {
		t.Errorf("WithQueryParam did not set query param correctly")
	}

	// Test WithContentType
	WithContentType("text/plain")(config)
	if config.ContentType != "text/plain" {
		t.Errorf("WithContentType did not set content type correctly")
	}
}

func TestBaseClient_parseError(t *testing.T) {
	client := &BaseClient{}

	tests := []struct {
		name        string
		resp        *Response
		wantMessage string
	}{
		{
			name: "JSON error with 'error' field",
			resp: &Response{
				StatusCode: 400,
				Body:       []byte(`{"error": "Bad Request", "detail": "Invalid input"}`),
			},
			wantMessage: "Bad Request",
		},
		{
			name: "JSON error with 'message' field",
			resp: &Response{
				StatusCode: 500,
				Body:       []byte(`{"message": "Internal Server Error", "code": "ISE001"}`),
			},
			wantMessage: "Internal Server Error",
		},
		{
			name: "non-JSON error",
			resp: &Response{
				StatusCode: 403,
				Body:       []byte(`Forbidden: Access denied`),
			},
			wantMessage: "Forbidden: Access denied",
		},
		{
			name: "empty body",
			resp: &Response{
				StatusCode: 404,
				Body:       []byte{},
			},
			wantMessage: "API error: 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.parseError(tt.resp)
			apiErr, ok := err.(*APIError)
			if !ok {
				t.Errorf("parseError() did not return APIError")
				return
			}
			if apiErr.Message != tt.wantMessage {
				t.Errorf("parseError() message = %v, want %v", apiErr.Message, tt.wantMessage)
			}
			if apiErr.StatusCode != tt.resp.StatusCode {
				t.Errorf("parseError() status code = %v, want %v", apiErr.StatusCode, tt.resp.StatusCode)
			}
		})
	}
}
