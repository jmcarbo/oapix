# OAPIx - OpenAPI Code Generator and Client Library for Go

OAPIx is a powerful Go library for generating type-safe clients from OpenAPI specifications, with built-in support for authentication, proxy configuration, and comprehensive error handling.

## Features

- **Code Generation**: Generate type-safe Go clients from OpenAPI 3.0 specifications
- **Clean Interface Design**: Modular architecture with reusable components
- **Multi-Response Handling**: Type-safe support for APIs with multiple response types per endpoint
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

#### Using Go Install

```bash
go install github.com/jmcarbo/oapix/cmd/oapix-gen@latest
```

#### Using Docker

```bash
# Build the Docker image
docker build -t oapix-gen .

# Or pull from Docker Hub (if published)
docker pull jmcarbo/oapix-gen:latest
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

#### Using Docker

```bash
# Generate client using Docker (mount current directory)
docker run --rm -v $(pwd):/work -w /work oapix-gen \
  -spec api.yaml \
  -package myapi \
  -output ./generated

# With custom options
docker run --rm -v $(pwd):/work -w /work oapix-gen \
  -spec api.yaml \
  -package myapi \
  -output ./generated \
  -client-import github.com/myorg/myclient \
  -verbose

# Using a spec from URL (requires network access)
docker run --rm -v $(pwd):/output -w /output oapix-gen \
  -spec https://api.example.com/openapi.yaml \
  -package myapi
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

## Multi-Response Handling

OAPIx provides comprehensive support for OpenAPI operations that can return different response types based on status codes. This is common in REST APIs where different status codes return different response schemas.

### Understanding Multi-Response APIs

Many REST APIs return different response types for different scenarios:
- **200 OK**: Returns the requested resource
- **201 Created**: Returns the newly created resource
- **204 No Content**: Success with no response body
- **400 Bad Request**: Returns validation errors
- **404 Not Found**: Returns a not-found error
- **500 Internal Server Error**: Returns server error details

### Basic Multi-Response Usage

When an operation has multiple possible response types, OAPIx generates methods that return a `*client.MultiResponse`:

```go
// API operation that can return different responses
resp, err := apiClient.CreateAsset(ctx, "asset-id", assetInput)
if err != nil {
    // Handle network or request building errors
    log.Fatal(err)
}

// Check status code to determine response type
switch resp.StatusCode {
case 200:
    // Asset was updated
    var asset Asset
    if err := resp.As(&asset); err != nil {
        log.Printf("Failed to parse asset: %v", err)
    }
    fmt.Printf("Asset updated: %s\n", asset.Name)

case 201:
    // Asset was created
    var asset Asset
    if err := resp.As(&asset); err != nil {
        log.Printf("Failed to parse asset: %v", err)
    }
    fmt.Printf("Asset created: %s\n", asset.Name)

case 400:
    // Bad request error
    var badRequest ErrorResponse
    if err := resp.As(&badRequest); err != nil {
        log.Printf("Failed to parse error: %v", err)
    }
    fmt.Printf("Validation error: %s\n", badRequest.Message)

default:
    fmt.Printf("Unexpected status: %d\n", resp.StatusCode)
}
```

### Helper Methods

The `MultiResponse` type provides convenient helper methods:

```go
// Check if response is successful (2xx)
if resp.IsSuccess() {
    var result SuccessResponse
    resp.As(&result)
}

// Check if response is an error (4xx or 5xx)
if resp.IsError() {
    fmt.Printf("Error response: %d\n", resp.StatusCode)
}

// Check for specific status code
if resp.Is(404) {
    var notFound NotFoundError
    resp.As(&notFound)
    fmt.Printf("Resource not found: %s\n", notFound.Resource)
}
```

### Type-Safe Response Wrappers

For operations with multiple success responses (e.g., both 200 and 201), OAPIx can generate type-safe wrapper methods:

```go
// Wrap the response for type-safe access
assetResp := WrapCreateAssetResponse(resp)

// Use type-safe methods for each status code
if assetResp.Is(200) {
    asset, err := assetResp.As200() // Returns (*Asset, error)
    if err != nil {
        log.Printf("Failed to get asset: %v", err)
        return
    }
    fmt.Printf("Updated asset: %s\n", asset.Name)
} else if assetResp.Is(201) {
    asset, err := assetResp.As201() // Returns (*Asset, error)
    if err != nil {
        log.Printf("Failed to get asset: %v", err)
        return
    }
    fmt.Printf("Created asset: %s\n", asset.Name)
}
```

### OpenAPI Specification Example

Here's an example OpenAPI specification with multiple response types:

```yaml
paths:
  /assets/{assetId}:
    post:
      operationId: createAsset
      parameters:
        - name: assetId
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/AssetInput'
      responses:
        "200":
          description: Asset updated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Asset'
        "201":
          description: Asset created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Asset'
        "400":
          description: Bad request
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
        "404":
          description: Not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/NotFoundError'
```

### Best Practices

1. **Always check the status code** before attempting to parse the response body
2. **Use helper methods** (`IsSuccess()`, `IsError()`, `Is()`) for cleaner code
3. **Handle all documented response types** in your OpenAPI spec
4. **Use type-safe wrappers** when available for compile-time safety
5. **Log unexpected status codes** to help with debugging

### Complete Example

See the [examples/multi_response_example.go](examples/multi_response_example.go) for a complete working example demonstrating all multi-response handling patterns.

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