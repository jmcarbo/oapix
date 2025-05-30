package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// BaseClient implements the base functionality for API clients
type BaseClient struct {
	httpClient     HTTPClient
	baseURL        string
	apiKey         string
	requestEditors []RequestEditor
}

// Config holds configuration for creating a new client
type Config struct {
	// BaseURL is the base URL of the API
	BaseURL string
	// APIKey for authentication (optional)
	APIKey string
	// HTTPClient is a custom HTTP client (optional)
	HTTPClient HTTPClient
	// TransportConfig for creating a default HTTP client (optional)
	TransportConfig *TransportConfig
	// RequestEditors are applied to all requests
	RequestEditors []RequestEditor
}

// NewBaseClient creates a new base client with the given configuration
func NewBaseClient(config *Config) (*BaseClient, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	// Ensure base URL ends with /
	if !strings.HasSuffix(config.BaseURL, "/") {
		config.BaseURL += "/"
	}

	// Use provided HTTP client or create a new one
	httpClient := config.HTTPClient
	if httpClient == nil {
		var err error
		httpClient, err = NewHTTPClient(config.TransportConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP client: %w", err)
		}
	}

	return &BaseClient{
		httpClient:     httpClient,
		baseURL:        config.BaseURL,
		apiKey:         config.APIKey,
		requestEditors: config.RequestEditors,
	}, nil
}

// Request makes an HTTP request with the given parameters
func (c *BaseClient) Request(ctx context.Context, method, path string, body io.Reader, opts ...RequestOption) (*Response, error) {
	// Apply request options
	config := &RequestConfig{}
	for _, opt := range opts {
		opt(config)
	}

	// Build full URL
	fullURL, err := c.buildURL(path, config.QueryParams)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Accept", "application/json")
	if config.ContentType != "" {
		req.Header.Set("Content-Type", config.ContentType)
	} else if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set API key if provided
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	// Apply custom headers
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	// Apply request editors
	for _, editor := range c.requestEditors {
		if err := editor(ctx, req); err != nil {
			return nil, fmt.Errorf("request editor failed: %w", err)
		}
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Create response
	response := &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return response, c.parseError(response)
	}

	return response, nil
}

// BaseURL returns the base URL of the API
func (c *BaseClient) BaseURL() string {
	return c.baseURL
}

// SetBaseURL sets the base URL of the API
func (c *BaseClient) SetBaseURL(baseURL string) {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	c.baseURL = baseURL
}

// AddRequestEditor adds a new request editor to the client
func (c *BaseClient) AddRequestEditor(editor RequestEditor) {
	c.requestEditors = append(c.requestEditors, editor)
}

// SetRequestEditors replaces all request editors
func (c *BaseClient) SetRequestEditors(editors []RequestEditor) {
	c.requestEditors = editors
}

// buildURL builds the full URL with query parameters
func (c *BaseClient) buildURL(path string, queryParams map[string]string) (string, error) {
	// Remove leading slash from path if present
	path = strings.TrimPrefix(path, "/")

	// Parse base URL
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Parse path
	pathURL, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Resolve path against base URL
	fullURL := baseURL.ResolveReference(pathURL)

	// Add query parameters
	if len(queryParams) > 0 {
		q := fullURL.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		fullURL.RawQuery = q.Encode()
	}

	return fullURL.String(), nil
}

// parseError parses an error response
func (c *BaseClient) parseError(resp *Response) error {
	apiError := &APIError{
		StatusCode: resp.StatusCode,
		Message:    fmt.Sprintf("API error: %d", resp.StatusCode),
	}

	// Try to parse JSON error response
	if len(resp.Body) > 0 {
		var errorData map[string]interface{}
		if err := json.Unmarshal(resp.Body, &errorData); err == nil {
			apiError.Details = errorData

			// Try to extract error message
			if msg, ok := errorData["error"].(string); ok {
				apiError.Message = msg
			} else if msg, ok := errorData["message"].(string); ok {
				apiError.Message = msg
			}
		} else {
			// If not JSON, use body as message
			apiError.Message = string(resp.Body)
		}
	}

	return apiError
}

// RequestJSON makes a JSON request
func (c *BaseClient) RequestJSON(ctx context.Context, method, path string, body interface{}, opts ...RequestOption) (*Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Ensure content type is set to JSON
	opts = append(opts, WithContentType("application/json"))

	return c.Request(ctx, method, path, bodyReader, opts...)
}

// ParseJSON parses a JSON response
func ParseJSON(resp *Response, v interface{}) error {
	if len(resp.Body) == 0 {
		return nil
	}

	if err := json.Unmarshal(resp.Body, v); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}
