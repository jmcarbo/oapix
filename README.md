# OAPIx - OpenAPI Code Generator and Client Library for Go

OAPIx is a powerful Go library for generating type-safe clients from OpenAPI specifications, with built-in support for authentication, proxy configuration, and comprehensive error handling.

## Features

- **Code Generation**: Generate type-safe Go clients from OpenAPI 3.0 specifications
- **Clean Interface Design**: Modular architecture with reusable components
- **Authentication Support**: Built-in OAuth2, API Key, Bearer token, and Basic auth
- **SOCKS Proxy Support**: Built-in support for SOCKS4/4a/5 proxies with authentication
- **Request Editors**: Dynamic request modification for headers, authentication, and more
- **Comprehensive Error Handling**: Structured error types with detailed API error information
- **Type-Safe Models**: Generated from OpenAPI specifications
- **Context Support**: Full context.Context support for cancellation and timeouts
- **Template Customization**: Override default templates with your own

## Installation

```bash
go get github.com/jmcarbo/oapix
```

## Code Generation

### Install the Generator

```bash
go install github.com/jmcarbo/oapix/cmd/oapix-gen@latest
```

### Generate a Client

```bash
# Generate client from OpenAPI spec
oapix-gen -spec api.yaml -out ./generated

# With custom package name
oapix-gen -spec api.yaml -out ./generated -package myapi

# With custom templates
oapix-gen -spec api.yaml -out ./generated -templates ./my-templates

# With custom client import path (use your own client implementation)
oapix-gen -spec api.yaml -out ./generated -package myapi -client-import github.com/myorg/myclient
```

### Generator Options

- `-spec` (required): Path to OpenAPI specification file
- `-out`: Output directory for generated code (default: current directory)
- `-package` (required): Package name for generated code
- `-client`: Name of the generated client struct (default: "Client")
- `-templates`: Directory containing custom templates
- `-model-package`: Package name for models (defaults to main package)
- `-client-package`: Package name for client (defaults to main package)
- `-client-import`: Custom import path for client packages (default: "github.com/jmcarbo/oapix/pkg/client")
- `-models-only`: Generate only models
- `-client-only`: Generate only client
- `-verbose`: Enable verbose output

### Custom Client Import

The `-client-import` flag allows you to use a custom client implementation instead of the default oapix client. This is useful when:

1. **Using a forked version**: You've forked oapix and want to use your customized client
   ```bash
   oapix-gen -spec api.yaml -package myapi -client-import github.com/myorg/oapix-fork/pkg/client
   ```

2. **Using a vendored client**: You've vendored the client packages in your project
   ```bash
   oapix-gen -spec api.yaml -package myapi -client-import myproject/vendor/oapix/client
   ```

3. **Using a completely custom implementation**: You've implemented your own client that follows the oapix interface
   ```bash
   oapix-gen -spec api.yaml -package myapi -client-import github.com/myorg/custom-http-client
   ```

The custom client package must implement the same interfaces and types as the original oapix client package.

**Note**: When using `-client-import`, the generated code will import from your specified path instead of the default:
```go
// Default import
import "github.com/jmcarbo/oapix/pkg/client"

// With -client-import github.com/myorg/myclient
import "github.com/myorg/myclient"
```

### Example OpenAPI Spec

```yaml
openapi: 3.0.0
info:
  title: Example API
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
paths:
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: User found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        email:
          type: string
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "github.com/jmcarbo/oapix/pkg/client"
    "github.com/jmcarbo/oapix/pkg/client/oauth2"
    "github.com/yourorg/generated" // Your generated client
)

func main() {
    // Create client configuration
    config := &client.Config{
        BaseURL: "https://api.example.com/v1",
        HTTPClient: &http.Client{
            Timeout: 30 * time.Second,
        },
        RequestEditors: []client.RequestEditor{
            // Add authentication
            oauth2.ClientCredentialsAuth(oauth2.ClientCredentialsConfig{
                ClientID:     "your-client-id",
                ClientSecret: "your-client-secret",
                TokenURL:     "https://auth.example.com/token",
                Scopes:       []string{"read", "write"},
            }),
        },
    }
    
    // Create API client
    apiClient, err := generated.NewClient(config)
    if err != nil {
        log.Fatal(err)
    }
    
    // Use the client
    ctx := context.Background()
    user, err := apiClient.GetUser(ctx, "user-123")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User: %s\n", user.Name)
}
```

## Authentication

### OAuth2 Client Credentials

```go
import "github.com/jmcarbo/oapix/pkg/client/oauth2"

config := &client.Config{
    BaseURL: "https://api.example.com",
    RequestEditors: []client.RequestEditor{
        oauth2.ClientCredentialsAuth(oauth2.ClientCredentialsConfig{
            ClientID:     "client-id",
            ClientSecret: "client-secret",
            TokenURL:     "https://auth.example.com/token",
            Scopes:       []string{"api.read", "api.write"},
        }),
    },
}
```

### API Key Authentication

```go
config := &client.Config{
    BaseURL: "https://api.example.com",
    RequestEditors: []client.RequestEditor{
        func(ctx context.Context, req *http.Request) error {
            req.Header.Set("X-API-Key", "your-api-key")
            return nil
        },
    },
}
```

### Bearer Token

```go
config := &client.Config{
    BaseURL: "https://api.example.com",
    RequestEditors: []client.RequestEditor{
        func(ctx context.Context, req *http.Request) error {
            req.Header.Set("Authorization", "Bearer your-token")
            return nil
        },
    },
}
```

## Using SOCKS Proxy

```go
import "golang.org/x/net/proxy"

// Create SOCKS5 dialer
dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:1080", 
    &proxy.Auth{User: "username", Password: "password"}, 
    proxy.Direct)
if err != nil {
    log.Fatal(err)
}

// Create HTTP client with SOCKS proxy
httpClient := &http.Client{
    Transport: &http.Transport{
        Dial: dialer.Dial,
    },
    Timeout: 30 * time.Second,
}

config := &client.Config{
    BaseURL:    "https://api.example.com",
    HTTPClient: httpClient,
}
```

## Error Handling

```go
user, err := apiClient.GetUser(ctx, "user-123")
if err != nil {
    var apiErr *client.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("API Error %d: %s\n", apiErr.StatusCode, apiErr.Message)
        fmt.Printf("Details: %+v\n", apiErr.Details)
    } else {
        fmt.Printf("Network error: %v\n", err)
    }
}
```

## Request Editors

RequestEditors allow you to modify requests before they are sent:

```go
// Add custom headers
customHeaders := func(ctx context.Context, req *http.Request) error {
    req.Header.Set("X-Custom-Header", "value")
    req.Header.Set("X-Request-ID", uuid.New().String())
    return nil
}

// Add user agent
userAgent := func(ctx context.Context, req *http.Request) error {
    req.Header.Set("User-Agent", "my-app/1.0")
    return nil
}

// Combine multiple editors
config := &client.Config{
    BaseURL: "https://api.example.com",
    RequestEditors: []client.RequestEditor{
        customHeaders,
        userAgent,
    },
}
```

## Advanced Features

### Custom HTTP Client

```go
// Create custom HTTP client with retry logic
httpClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &retryablehttp.RoundTripper{
        Client: retryablehttp.NewClient(),
    },
}

config := &client.Config{
    BaseURL:    "https://api.example.com",
    HTTPClient: httpClient,
}
```

### Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

user, err := apiClient.GetUser(ctx, "user-123")
```

### Request Options

```go
// If your generated client supports request options
resp, err := apiClient.ListUsers(ctx, 
    WithQueryParam("page", "2"),
    WithQueryParam("limit", "50"),
    WithHeader("X-Debug", "true"),
)
```

## Testing

### Unit Tests

```bash
go test ./...
```

### Integration Tests

```bash
# Run with integration tag
go test -tags=integration ./...

# With environment variables
API_BASE_URL=https://api.example.com \
API_KEY=your-key \
go test -tags=integration ./...
```

### Mocking

The generated clients use interfaces, making them easy to mock:

```go
type MockClient struct {
    mock.Mock
}

func (m *MockClient) GetUser(ctx context.Context, id string) (*User, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*User), args.Error(1)
}
```

## Project Structure

```
oapix/
├── cmd/
│   └── oapix-gen/        # Code generator CLI
├── pkg/
│   ├── gen/              # Code generation logic
│   │   ├── generator.go  # Main generator
│   │   └── templates/    # Go templates
│   ├── client/           # Base client functionality
│   │   ├── interface.go  # Client interfaces
│   │   ├── base.go       # Base implementation
│   │   └── oauth2/       # OAuth2 support
│   └── example/          # Example generated client
├── examples/             # Example OpenAPI specs
└── README.md
```

## Customizing Templates

You can override the default templates by providing your own:

1. Create a directory with your custom templates
2. Use the same filenames as the default templates:
   - `client.tmpl` - Client implementation
   - `models.tmpl` - Model definitions
   - `interface.tmpl` - Client interface

3. Run the generator with your templates:
```bash
oapix-gen -spec api.yaml -out ./generated -templates ./my-templates
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.