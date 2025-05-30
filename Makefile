.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: fmt
fmt: ## Format code using gofumpt
	gofumpt -l -w .

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -cover ./...

.PHONY: test-coverage-html
test-coverage-html: ## Generate HTML coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: build
build: ## Build the project
	go build -o bin/oapix

.PHONY: build-gen
build-gen: ## Build the oapix-gen code generator
	go build -o bin/oapix-gen ./cmd/oapix-gen/

.PHONY: build-all
build-all: build build-gen ## Build all binaries

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf tmp/
	rm -f coverage.out coverage.html

.PHONY: deps
deps: ## Download and tidy dependencies
	go mod download
	go mod tidy

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: check
check: fmt lint vet test ## Run all checks (format, lint, vet, test)

.PHONY: install-tools
install-tools: ## Install required development tools
	go install mvdan.cc/gofumpt@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: run
run: ## Run the application
	go run .

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t oapix:latest .

.PHONY: all
all: clean deps fmt lint test build ## Run full build pipeline