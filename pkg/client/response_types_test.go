package client

import (
	"encoding/json"
	"testing"
)

func TestMultiResponse_As(t *testing.T) {
	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	tests := []struct {
		name     string
		response Response
		target   interface{}
		wantErr  bool
	}{
		{
			name: "successful json unmarshal",
			response: Response{
				StatusCode: 200,
				Body:       []byte(`{"name":"test","value":42}`),
			},
			target:  &TestData{},
			wantErr: false,
		},
		{
			name: "invalid json",
			response: Response{
				StatusCode: 200,
				Body:       []byte(`{"invalid json`),
			},
			target:  &TestData{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &MultiResponse{
				Response: tt.response,
			}

			err := mr.As(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("MultiResponse.As() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.name == "successful json unmarshal" {
				data := tt.target.(*TestData)
				if data.Name != "test" || data.Value != 42 {
					t.Errorf("MultiResponse.As() = %+v, want {Name:test Value:42}", data)
				}
			}
		})
	}
}

func TestMultiResponse_Is(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		checkCode  int
		want       bool
	}{
		{
			name:       "matching status code",
			statusCode: 200,
			checkCode:  200,
			want:       true,
		},
		{
			name:       "non-matching status code",
			statusCode: 200,
			checkCode:  404,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &MultiResponse{
				Response: Response{
					StatusCode: tt.statusCode,
				},
			}

			if got := mr.Is(tt.checkCode); got != tt.want {
				t.Errorf("MultiResponse.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiResponse_IsSuccess(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{
			name:       "200 OK",
			statusCode: 200,
			want:       true,
		},
		{
			name:       "201 Created",
			statusCode: 201,
			want:       true,
		},
		{
			name:       "204 No Content",
			statusCode: 204,
			want:       true,
		},
		{
			name:       "299 edge case",
			statusCode: 299,
			want:       true,
		},
		{
			name:       "300 redirect",
			statusCode: 300,
			want:       false,
		},
		{
			name:       "400 bad request",
			statusCode: 400,
			want:       false,
		},
		{
			name:       "500 server error",
			statusCode: 500,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &MultiResponse{
				Response: Response{
					StatusCode: tt.statusCode,
				},
			}

			if got := mr.IsSuccess(); got != tt.want {
				t.Errorf("MultiResponse.IsSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiResponse_IsError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       bool
	}{
		{
			name:       "200 OK",
			statusCode: 200,
			want:       false,
		},
		{
			name:       "300 redirect",
			statusCode: 300,
			want:       false,
		},
		{
			name:       "400 bad request",
			statusCode: 400,
			want:       true,
		},
		{
			name:       "404 not found",
			statusCode: 404,
			want:       true,
		},
		{
			name:       "500 server error",
			statusCode: 500,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mr := &MultiResponse{
				Response: Response{
					StatusCode: tt.statusCode,
				},
			}

			if got := mr.IsError(); got != tt.want {
				t.Errorf("MultiResponse.IsError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiResponse_TypeAssertion(t *testing.T) {
	// Test the type assertion functionality
	rawData := json.RawMessage(`{"test": "value"}`)
	mr := &MultiResponse{
		Response: Response{
			StatusCode: 200,
		},
		parsedBody: &rawData,
	}

	var result *json.RawMessage
	err := mr.As(&result)
	if err != nil {
		t.Errorf("MultiResponse.As() with RawMessage failed: %v", err)
	}

	if result == nil || string(*result) != string(rawData) {
		t.Errorf("MultiResponse.As() with RawMessage = %v, want %v", result, rawData)
	}
}

// MockHTTPResponse creates a mock Response for testing
func MockHTTPResponse(statusCode int, body string, headers map[string][]string) Response {
	if headers == nil {
		headers = make(map[string][]string)
	}
	headers["Content-Type"] = []string{"application/json"}

	return Response{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       []byte(body),
	}
}

func TestMultiResponseIntegration(t *testing.T) {
	// Test real-world scenario with different response types
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	type Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	// Simulate 200 response
	successResp := MockHTTPResponse(200, `{"id":1,"name":"John Doe"}`, nil)
	mr := &MultiResponse{Response: successResp}

	if !mr.IsSuccess() {
		t.Error("Expected success response")
	}

	var user User
	if err := mr.As(&user); err != nil {
		t.Errorf("Failed to parse user: %v", err)
	}

	if user.ID != 1 || user.Name != "John Doe" {
		t.Errorf("Unexpected user data: %+v", user)
	}

	// Simulate 404 response
	errorResp := MockHTTPResponse(404, `{"code":"NOT_FOUND","message":"User not found"}`, nil)
	mr = &MultiResponse{Response: errorResp}

	if !mr.IsError() {
		t.Error("Expected error response")
	}

	var apiError Error
	if err := mr.As(&apiError); err != nil {
		t.Errorf("Failed to parse error: %v", err)
	}

	if apiError.Code != "NOT_FOUND" || apiError.Message != "User not found" {
		t.Errorf("Unexpected error data: %+v", apiError)
	}
}
