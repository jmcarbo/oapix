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

# Docker registry configuration
DOCKER_REGISTRY ?= ghcr.io
DOCKER_ORG ?= jmcarbo
DOCKER_IMAGE ?= $(DOCKER_REGISTRY)/$(DOCKER_ORG)/oapix
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
VERSION_NO_V := $(shell echo $(VERSION) | sed 's/^v//')

.PHONY: docker-build
docker-build: ## Build Docker image with version tag
	docker build -t $(DOCKER_IMAGE):$(VERSION_NO_V) -t $(DOCKER_IMAGE):latest .

.PHONY: docker-build-all
docker-build-all: docker-build ## Build and tag Docker image with multiple tags
	@if echo "$(VERSION)" | grep -qE "^v[0-9]+\.[0-9]+\.0$$"; then \
		MINOR=$$(echo $(VERSION_NO_V) | cut -d. -f1,2); \
		docker tag $(DOCKER_IMAGE):$(VERSION_NO_V) $(DOCKER_IMAGE):$$MINOR; \
		echo "Tagged as $(DOCKER_IMAGE):$$MINOR"; \
	fi
	@if echo "$(VERSION)" | grep -qE "^v[0-9]+\.0\.0$$"; then \
		MAJOR=$$(echo $(VERSION_NO_V) | cut -d. -f1); \
		docker tag $(DOCKER_IMAGE):$(VERSION_NO_V) $(DOCKER_IMAGE):$$MAJOR; \
		echo "Tagged as $(DOCKER_IMAGE):$$MAJOR"; \
	fi

.PHONY: docker-push
docker-push: docker-build-all ## Build and push Docker image with all tags
	docker push $(DOCKER_IMAGE):$(VERSION_NO_V)
	docker push $(DOCKER_IMAGE):latest
	@if echo "$(VERSION)" | grep -qE "^v[0-9]+\.[0-9]+\.0$$"; then \
		MINOR=$$(echo $(VERSION_NO_V) | cut -d. -f1,2); \
		docker push $(DOCKER_IMAGE):$$MINOR; \
	fi
	@if echo "$(VERSION)" | grep -qE "^v[0-9]+\.0\.0$$"; then \
		MAJOR=$$(echo $(VERSION_NO_V) | cut -d. -f1); \
		docker push $(DOCKER_IMAGE):$$MAJOR; \
	fi

.PHONY: docker-info
docker-info: ## Show Docker image information
	@echo "Docker image: $(DOCKER_IMAGE)"
	@echo "Version: $(VERSION) ($(VERSION_NO_V))"
	@echo "Tags that would be created:"
	@echo "  - $(DOCKER_IMAGE):$(VERSION_NO_V)"
	@echo "  - $(DOCKER_IMAGE):latest"
	@if echo "$(VERSION)" | grep -qE "^v[0-9]+\.[0-9]+\.0$$"; then \
		MINOR=$$(echo $(VERSION_NO_V) | cut -d. -f1,2); \
		echo "  - $(DOCKER_IMAGE):$$MINOR"; \
	fi
	@if echo "$(VERSION)" | grep -qE "^v[0-9]+\.0\.0$$"; then \
		MAJOR=$$(echo $(VERSION_NO_V) | cut -d. -f1); \
		echo "  - $(DOCKER_IMAGE):$$MAJOR"; \
	fi

.PHONY: all
all: clean deps fmt lint test build ## Run full build pipeline

.PHONY: version
version: ## Show current version
	@git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"