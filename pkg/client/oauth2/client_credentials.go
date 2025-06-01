package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jmcarbo/oapix/pkg/client"
)

// TokenResponse represents an OAuth2 access token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ClientCredentialsConfig holds configuration for OAuth2 client credentials flow
type ClientCredentialsConfig struct {
	// ClientID is the OAuth2 client ID
	ClientID string
	// ClientSecret is the OAuth2 client secret
	ClientSecret string
	// TokenURL is the OAuth2 token endpoint URL
	TokenURL string
	// Scopes is the list of OAuth2 scopes to request
	Scopes []string
	// HTTPClient is the HTTP client to use for token requests (optional)
	HTTPClient *http.Client
	// ExpiryDelta is the duration before token expiry to refresh (default: 60 seconds)
	ExpiryDelta time.Duration
}

// ClientCredentialsTokenSource implements OAuth2 client credentials flow
type ClientCredentialsTokenSource struct {
	config      ClientCredentialsConfig
	token       *TokenResponse
	tokenExpiry time.Time
	mu          sync.RWMutex
}

// NewClientCredentialsTokenSource creates a new OAuth2 client credentials token source
func NewClientCredentialsTokenSource(config ClientCredentialsConfig) *ClientCredentialsTokenSource {
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}
	if config.ExpiryDelta == 0 {
		config.ExpiryDelta = 60 * time.Second
	}
	return &ClientCredentialsTokenSource{
		config: config,
	}
}

// Token returns a valid OAuth2 token, refreshing if necessary
func (c *ClientCredentialsTokenSource) Token(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.token != nil && time.Now().Before(c.tokenExpiry) {
		defer c.mu.RUnlock()
		return c.token.AccessToken, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.token != nil && time.Now().Before(c.tokenExpiry) {
		return c.token.AccessToken, nil
	}

	// Request new token
	token, err := c.requestToken(ctx)
	if err != nil {
		return "", err
	}

	c.token = token
	// Set expiry with buffer time
	c.tokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn)*time.Second - c.config.ExpiryDelta)

	return token.AccessToken, nil
}

// requestToken requests a new OAuth2 token
func (c *ClientCredentialsTokenSource) requestToken(ctx context.Context) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.config.ClientID)
	data.Set("client_secret", c.config.ClientSecret)
	if len(c.config.Scopes) > 0 {
		data.Set("scope", strings.Join(c.config.Scopes, " "))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &token, nil
}

// ClientCredentialsAuth returns a RequestEditor that adds OAuth2 Bearer token authentication
// using the client credentials flow
func ClientCredentialsAuth(config ClientCredentialsConfig) client.RequestEditor {
	tokenSource := NewClientCredentialsTokenSource(config)
	return func(ctx context.Context, req *http.Request) error {
		token, err := tokenSource.Token(ctx)
		if err != nil {
			return fmt.Errorf("failed to get OAuth2 token: %w", err)
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return nil
	}
}
