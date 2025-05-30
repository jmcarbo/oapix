package client

import (
	"context"
	"io"
	"net/http"
)

// Client is the base interface for all API clients
type Client interface {
	// Request makes an HTTP request with the given parameters
	Request(ctx context.Context, method, path string, body io.Reader, opts ...RequestOption) (*Response, error)
	// RequestJSON makes a JSON request with the given parameters
	RequestJSON(ctx context.Context, method, path string, body interface{}, opts ...RequestOption) (*Response, error)
	// BaseURL returns the base URL of the API
	BaseURL() string
	// SetBaseURL sets the base URL of the API
	SetBaseURL(baseURL string)
}

// RequestEditor is a function that can modify an HTTP request before it's sent
type RequestEditor func(ctx context.Context, req *http.Request) error

// Response wraps the HTTP response with additional functionality
type Response struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

// RequestOption is a function that modifies a request
type RequestOption func(*RequestConfig)

// RequestConfig holds configuration for a single request
type RequestConfig struct {
	Headers     map[string]string
	QueryParams map[string]string
	ContentType string
}

// WithHeader adds a header to the request
func WithHeader(key, value string) RequestOption {
	return func(c *RequestConfig) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		c.Headers[key] = value
	}
}

// WithQueryParam adds a query parameter to the request
func WithQueryParam(key, value string) RequestOption {
	return func(c *RequestConfig) {
		if c.QueryParams == nil {
			c.QueryParams = make(map[string]string)
		}
		c.QueryParams[key] = value
	}
}

// WithContentType sets the content type of the request
func WithContentType(contentType string) RequestOption {
	return func(c *RequestConfig) {
		c.ContentType = contentType
	}
}

// APIError represents an API error response
type APIError struct {
	StatusCode int
	Message    string
	Details    interface{}
}

func (e *APIError) Error() string {
	return e.Message
}
