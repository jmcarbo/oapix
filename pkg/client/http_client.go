package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// HTTPClient is an interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// TransportConfig holds configuration for HTTP transport
type TransportConfig struct {
	// Timeout for the entire request
	Timeout time.Duration
	// SOCKS proxy configuration
	SOCKSProxy *SOCKSConfig
	// Custom headers to add to all requests
	Headers map[string]string
	// User agent string
	UserAgent string
}

// SOCKSConfig holds SOCKS proxy configuration
type SOCKSConfig struct {
	// Address of the SOCKS proxy (e.g., "127.0.0.1:1080")
	Address string
	// Username for SOCKS authentication (optional)
	Username string
	// Password for SOCKS authentication (optional)
	Password string
	// Version of SOCKS proxy (currently only 5 is supported)
	Version int
}

// NewHTTPClient creates a new HTTP client with the given configuration
func NewHTTPClient(config *TransportConfig) (HTTPClient, error) {
	if config == nil {
		config = &TransportConfig{
			Timeout: 30 * time.Second,
		}
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Configure SOCKS proxy if provided
	if config.SOCKSProxy != nil {
		dialer, err := createSOCKSDialer(config.SOCKSProxy)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS dialer: %w", err)
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	return &httpClientWrapper{
		client:    client,
		headers:   config.Headers,
		userAgent: config.UserAgent,
	}, nil
}

// httpClientWrapper wraps http.Client to add custom headers
type httpClientWrapper struct {
	client    *http.Client
	headers   map[string]string
	userAgent string
}

func (c *httpClientWrapper) Do(req *http.Request) (*http.Response, error) {
	// Add custom headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Set user agent if provided
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	return c.client.Do(req)
}

// createSOCKSDialer creates a SOCKS proxy dialer
func createSOCKSDialer(config *SOCKSConfig) (proxy.Dialer, error) {
	// Validate SOCKS version
	if config.Version != 5 {
		return nil, fmt.Errorf("unsupported SOCKS version %d, only SOCKS5 is supported", config.Version)
	}

	// Build SOCKS URL with authentication if provided
	var socksURLStr string
	if config.Username != "" && config.Password != "" {
		socksURLStr = fmt.Sprintf("socks%d://%s:%s@%s",
			config.Version,
			url.QueryEscape(config.Username),
			url.QueryEscape(config.Password),
			config.Address)
	} else {
		socksURLStr = fmt.Sprintf("socks%d://%s", config.Version, config.Address)
	}

	// Parse SOCKS URL
	socksURL, err := url.Parse(socksURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SOCKS address: %w", err)
	}

	dialer, err := proxy.FromURL(socksURL, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("failed to create SOCKS dialer: %w", err)
	}

	return dialer, nil
}
