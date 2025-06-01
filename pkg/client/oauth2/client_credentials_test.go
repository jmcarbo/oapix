package oauth2

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientCredentialsTokenSource(t *testing.T) {
	tokenCount := 0
	expectedToken := "test-access-token"
	expectedScopes := []string{"read", "write"}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}

		// Parse form data
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}

		// Verify parameters
		if got := r.Form.Get("grant_type"); got != "client_credentials" {
			t.Errorf("Expected grant_type=client_credentials, got %s", got)
		}
		if got := r.Form.Get("client_id"); got != "test-client-id" {
			t.Errorf("Expected client_id=test-client-id, got %s", got)
		}
		if got := r.Form.Get("client_secret"); got != "test-client-secret" {
			t.Errorf("Expected client_secret=test-client-secret, got %s", got)
		}
		if got := r.Form.Get("scope"); got != "read write" {
			t.Errorf("Expected scope='read write', got %s", got)
		}

		tokenCount++

		// Return token response
		resp := TokenResponse{
			AccessToken: expectedToken,
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			Scope:       strings.Join(expectedScopes, " "),
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	// Create token source
	config := ClientCredentialsConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     server.URL,
		Scopes:       expectedScopes,
		ExpiryDelta:  time.Second,
	}
	tokenSource := NewClientCredentialsTokenSource(config)

	ctx := context.Background()

	// First token request
	token1, err := tokenSource.Token(ctx)
	if err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}
	if token1 != expectedToken {
		t.Errorf("Expected token %s, got %s", expectedToken, token1)
	}
	if tokenCount != 1 {
		t.Errorf("Expected 1 token request, got %d", tokenCount)
	}

	// Second request should use cached token
	token2, err := tokenSource.Token(ctx)
	if err != nil {
		t.Fatalf("Failed to get cached token: %v", err)
	}
	if token2 != expectedToken {
		t.Errorf("Expected token %s, got %s", expectedToken, token2)
	}
	if tokenCount != 1 {
		t.Errorf("Expected 1 token request (cached), got %d", tokenCount)
	}

	// Force token expiry
	tokenSource.mu.Lock()
	tokenSource.tokenExpiry = time.Now().Add(-time.Minute)
	tokenSource.mu.Unlock()

	// Third request should fetch new token
	token3, err := tokenSource.Token(ctx)
	if err != nil {
		t.Fatalf("Failed to get new token: %v", err)
	}
	if token3 != expectedToken {
		t.Errorf("Expected token %s, got %s", expectedToken, token3)
	}
	if tokenCount != 2 {
		t.Errorf("Expected 2 token requests, got %d", tokenCount)
	}
}

func TestClientCredentialsAuth(t *testing.T) {
	expectedToken := "test-bearer-token"

	// Create test server for token endpoint
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TokenResponse{
			AccessToken: expectedToken,
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("Failed to encode response: %v", err)
		}
	}))
	defer tokenServer.Close()

	// Create RequestEditor
	editor := ClientCredentialsAuth(ClientCredentialsConfig{
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		TokenURL:     tokenServer.URL,
	})

	// Test the editor
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	ctx := context.Background()

	if err := editor(ctx, req); err != nil {
		t.Fatalf("Failed to edit request: %v", err)
	}

	authHeader := req.Header.Get("Authorization")
	expectedHeader := "Bearer " + expectedToken
	if authHeader != expectedHeader {
		t.Errorf("Expected Authorization header %s, got %s", expectedHeader, authHeader)
	}
}

func TestClientCredentialsTokenSource_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		serverHandler func(w http.ResponseWriter, r *http.Request)
		expectedError string
	}{
		{
			name: "HTTP 401 Unauthorized",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error": "invalid_client"}`))
			},
			expectedError: "token request failed with status 401",
		},
		{
			name: "Invalid JSON response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`invalid json`))
			},
			expectedError: "failed to decode token response",
		},
		{
			name: "Empty response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			},
			expectedError: "", // Should succeed but with empty token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverHandler))
			defer server.Close()

			tokenSource := NewClientCredentialsTokenSource(ClientCredentialsConfig{
				ClientID:     "test-client",
				ClientSecret: "test-secret",
				TokenURL:     server.URL,
			})

			ctx := context.Background()
			_, err := tokenSource.Token(ctx)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
