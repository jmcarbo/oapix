package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const (
	// contextKeyCancel is the context key for cancel functions
	contextKeyCancel contextKey = "cancel"
	// contextKeyRetryConfig is the context key for retry configuration
	contextKeyRetryConfig contextKey = "retryConfig"
)

// Common RequestEditor implementations

// BearerTokenAuth returns a RequestEditor that adds Bearer token authentication
func BearerTokenAuth(token string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

// BasicAuth returns a RequestEditor that adds Basic authentication
func BasicAuth(username, password string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		auth := username + ":" + password
		encoded := base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", "Basic "+encoded)
		return nil
	}
}

// APIKeyAuth returns a RequestEditor that adds API key authentication
func APIKeyAuth(headerName, apiKey string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set(headerName, apiKey)
		return nil
	}
}

// QueryParamAuth returns a RequestEditor that adds authentication via query parameter
func QueryParamAuth(paramName, value string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		q := req.URL.Query()
		q.Set(paramName, value)
		req.URL.RawQuery = q.Encode()
		return nil
	}
}

// CustomHeader returns a RequestEditor that adds a custom header
func CustomHeader(name, value string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set(name, value)
		return nil
	}
}

// CustomHeaders returns a RequestEditor that adds multiple custom headers
func CustomHeaders(headers map[string]string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		return nil
	}
}

// UserAgent returns a RequestEditor that sets the User-Agent header
func UserAgent(userAgent string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("User-Agent", userAgent)
		return nil
	}
}

// RequestID returns a RequestEditor that adds a request ID header
func RequestID(headerName string, generator func() string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		requestID := generator()
		req.Header.Set(headerName, requestID)
		return nil
	}
}

// Timeout returns a RequestEditor that sets a timeout on the request context
func Timeout(duration time.Duration) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		newCtx, cancel := context.WithTimeout(ctx, duration)
		// Store cancel function in context for potential cleanup
		*req = *req.WithContext(context.WithValue(newCtx, contextKeyCancel, cancel))
		return nil
	}
}

// OAuth2TokenSource represents a source of OAuth2 tokens
type OAuth2TokenSource interface {
	Token(ctx context.Context) (string, error)
}

// OAuth2Auth returns a RequestEditor that adds OAuth2 Bearer token authentication
// The token is fetched dynamically from the provided TokenSource
func OAuth2Auth(tokenSource OAuth2TokenSource) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		token, err := tokenSource.Token(ctx)
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}

// ChainRequestEditors chains multiple RequestEditors together
func ChainRequestEditors(editors ...RequestEditor) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		for _, editor := range editors {
			if err := editor(ctx, req); err != nil {
				return err
			}
		}
		return nil
	}
}

// ConditionalRequestEditor applies a RequestEditor based on a condition
func ConditionalRequestEditor(condition func(ctx context.Context, req *http.Request) bool, editor RequestEditor) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		if condition(ctx, req) {
			return editor(ctx, req)
		}
		return nil
	}
}

// RetryableRequestEditor marks a request as retryable with metadata
func RetryableRequestEditor(maxRetries int, retryableStatusCodes ...int) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		// Store retry configuration in context
		retryConfig := map[string]interface{}{
			"maxRetries":           maxRetries,
			"retryableStatusCodes": retryableStatusCodes,
		}
		*req = *req.WithContext(context.WithValue(req.Context(), contextKeyRetryConfig, retryConfig))
		return nil
	}
}

// SignedRequestEditor signs requests using a custom signing function
type RequestSigner interface {
	SignRequest(ctx context.Context, req *http.Request) error
}

// SignedRequest returns a RequestEditor that signs the request
func SignedRequest(signer RequestSigner) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		return signer.SignRequest(ctx, req)
	}
}

// DebugRequestEditor logs request details (useful for debugging)
type RequestLogger interface {
	LogRequest(ctx context.Context, req *http.Request)
}

// DebugRequest returns a RequestEditor that logs the request
func DebugRequest(logger RequestLogger) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		logger.LogRequest(ctx, req)
		return nil
	}
}

// CookieAuth returns a RequestEditor that adds cookie authentication
func CookieAuth(cookies ...*http.Cookie) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		return nil
	}
}

// RefererHeader returns a RequestEditor that sets the Referer header
func RefererHeader(referer string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Referer", referer)
		return nil
	}
}

// AcceptHeader returns a RequestEditor that sets the Accept header
func AcceptHeader(mediaType string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Accept", mediaType)
		return nil
	}
}

// AcceptLanguage returns a RequestEditor that sets the Accept-Language header
func AcceptLanguage(language string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Accept-Language", language)
		return nil
	}
}

// XForwardedFor returns a RequestEditor that sets the X-Forwarded-For header
func XForwardedFor(ip string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("X-Forwarded-For", ip)
		return nil
	}
}

// ContentType returns a RequestEditor that sets the Content-Type header
func ContentType(mediaType string) RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Content-Type", mediaType)
		return nil
	}
}
