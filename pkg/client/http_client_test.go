package client

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *TransportConfig
		wantErr bool
	}{
		{
			name:    "nil config uses defaults",
			config:  nil,
			wantErr: false,
		},
		{
			name: "basic config",
			config: &TransportConfig{
				Timeout: 60 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "config with headers",
			config: &TransportConfig{
				Headers: map[string]string{
					"X-API-Key": "test-key",
				},
				UserAgent: "test-agent/1.0",
			},
			wantErr: false,
		},
		{
			name: "config with SOCKS proxy",
			config: &TransportConfig{
				SOCKSProxy: &SOCKSConfig{
					Address: "127.0.0.1:1080",
					Version: 5,
				},
			},
			wantErr: false,
		},
		{
			name: "config with SOCKS proxy and auth",
			config: &TransportConfig{
				SOCKSProxy: &SOCKSConfig{
					Address:  "127.0.0.1:1080",
					Username: "user",
					Password: "pass",
					Version:  5,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewHTTPClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHTTPClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewHTTPClient() returned nil client")
			}
		})
	}
}

func TestHTTPClientWrapper_Do(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back headers for testing
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Check custom headers
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			_, _ = w.Write([]byte(`{"api_key": "` + apiKey + `"}`))
		} else if userAgent := r.Header.Get("User-Agent"); userAgent != "" {
			_, _ = w.Write([]byte(`{"user_agent": "` + userAgent + `"}`))
		} else {
			_, _ = w.Write([]byte(`{"status": "ok"}`))
		}
	}))
	defer server.Close()

	tests := []struct {
		name      string
		config    *TransportConfig
		checkFunc func(t *testing.T, resp *http.Response, body string)
	}{
		{
			name: "request with custom headers",
			config: &TransportConfig{
				Headers: map[string]string{
					"X-API-Key": "test-key-123",
				},
			},
			checkFunc: func(t *testing.T, resp *http.Response, body string) {
				if !strings.Contains(body, "test-key-123") {
					t.Errorf("Expected body to contain API key, got: %s", body)
				}
			},
		},
		{
			name: "request with user agent",
			config: &TransportConfig{
				UserAgent: "test-client/1.0",
			},
			checkFunc: func(t *testing.T, resp *http.Response, body string) {
				if !strings.Contains(body, "test-client/1.0") {
					t.Errorf("Expected body to contain user agent, got: %s", body)
				}
			},
		},
		{
			name: "request with timeout",
			config: &TransportConfig{
				Timeout: 5 * time.Second,
			},
			checkFunc: func(t *testing.T, resp *http.Response, body string) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Expected status OK, got: %d", resp.StatusCode)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewHTTPClient(tt.config)
			if err != nil {
				t.Fatalf("Failed to create HTTP client: %v", err)
			}

			req, err := http.NewRequest("GET", server.URL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer func() {
				_ = resp.Body.Close()
			}()

			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			bodyStr := string(body[:n])

			tt.checkFunc(t, resp, bodyStr)
		})
	}
}

func TestCreateSOCKSDialer(t *testing.T) {
	tests := []struct {
		name    string
		config  *SOCKSConfig
		wantErr bool
	}{
		{
			name: "SOCKS5 without auth",
			config: &SOCKSConfig{
				Address: "127.0.0.1:1080",
				Version: 5,
			},
			wantErr: false,
		},
		{
			name: "SOCKS5 with auth",
			config: &SOCKSConfig{
				Address:  "127.0.0.1:1080",
				Username: "user",
				Password: "pass",
				Version:  5,
			},
			wantErr: false,
		},
		{
			name: "SOCKS4",
			config: &SOCKSConfig{
				Address: "127.0.0.1:1080",
				Version: 4,
			},
			wantErr: true, // golang.org/x/net/proxy only supports SOCKS5
		},
		{
			name: "invalid address",
			config: &SOCKSConfig{
				Address: "invalid address",
				Version: 5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dialer, err := createSOCKSDialer(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("createSOCKSDialer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && dialer == nil {
				t.Error("createSOCKSDialer() returned nil dialer")
			}
		})
	}
}

func TestHTTPClientIntegration(t *testing.T) {
	// Skip if short test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/timeout":
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		case "/echo":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"path": "` + r.URL.Path + `", "method": "` + r.Method + `"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	t.Run("timeout handling", func(t *testing.T) {
		config := &TransportConfig{
			Timeout: 1 * time.Second,
		}
		client, err := NewHTTPClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		req, _ := http.NewRequest("GET", server.URL+"/timeout", nil)
		_, err = client.Do(req)
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}

		// Check if it's a timeout error
		if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
			t.Errorf("Expected timeout error, got: %v", err)
		}
	})

	t.Run("successful request", func(t *testing.T) {
		config := &TransportConfig{
			Timeout: 5 * time.Second,
		}
		client, err := NewHTTPClient(config)
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}

		req, _ := http.NewRequest("GET", server.URL+"/echo", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got: %d", resp.StatusCode)
		}
	})
}

func TestHTTPClientWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	client, err := NewHTTPClient(nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Cancel context immediately
		cancel()

		_, err = client.Do(req)
		if err == nil {
			t.Error("Expected context cancellation error, got nil")
		}
	})

	t.Run("context with deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		_, err = client.Do(req)
		if err == nil {
			t.Error("Expected deadline exceeded error, got nil")
		}
	})
}
