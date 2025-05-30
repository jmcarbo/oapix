# Build stage
FROM golang:1.22-alpine AS builder

# Install certificates for HTTPS support
RUN apk add --no-cache ca-certificates

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build static binary
# CGO_ENABLED=0 for static binary
# -ldflags for smaller binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o oapix-gen \
    ./cmd/oapix-gen

# Final stage
FROM scratch

# Copy SSL certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder
COPY --from=builder /build/oapix-gen /oapix-gen

# Set the binary as entrypoint
ENTRYPOINT ["/oapix-gen"]