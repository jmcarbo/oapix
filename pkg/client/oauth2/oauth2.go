// Package oauth2 provides OAuth2 authentication support for the oapix client
package oauth2

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jmcarbo/oapix/pkg/client"
)

// TokenSource is an interface for OAuth2 token sources
type TokenSource interface {
	// Token returns a valid OAuth2 token
	Token(ctx context.Context) (string, error)
}

// Auth returns a RequestEditor that adds OAuth2 Bearer token authentication
// The token is fetched dynamically from the provided TokenSource
func Auth(tokenSource TokenSource) client.RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		token, err := tokenSource.Token(ctx)
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 token: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return nil
	}
}

// StaticTokenSource provides a static OAuth2 token
type StaticTokenSource struct {
	token string
}

// NewStaticTokenSource creates a new static token source
func NewStaticTokenSource(token string) *StaticTokenSource {
	return &StaticTokenSource{token: token}
}

// Token returns the static token
func (s *StaticTokenSource) Token(ctx context.Context) (string, error) {
	if s.token == "" {
		return "", fmt.Errorf("static token is empty")
	}
	return s.token, nil
}

// StaticAuth returns a RequestEditor that adds a static Bearer token
func StaticAuth(token string) client.RequestEditor {
	return Auth(NewStaticTokenSource(token))
}