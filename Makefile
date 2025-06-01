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

.PHONY: docker-push
docker-push: docker-build ## Build and push Docker image to registry
	docker push oapix:latest

.PHONY: all
all: clean deps fmt lint test build ## Run full build pipeline

.PHONY: version
version: ## Show current version
	@git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"

.PHONY: version-bump
version-bump: ## Analyze commits and create new version tag
	@./scripts/version-bump.sh

.PHONY: version-bump-dry
version-bump-dry: ## Dry run of version bump (show what would happen)
	@DRY_RUN=true ./scripts/version-bump.sh

.PHONY: release
release: check version-bump ## Run checks and create a new release tag
	@echo "Release created. Don't forget to push the tag!"
	@echo "Run: git push origin $$(git describe --tags --abbrev=0)"

.PHONY: release-dry
release-dry: check version-bump-dry ## Dry run of release process