# GoLinks Makefile

# Variables
BINARY_NAME=golinks
BUILD_DIR=./build

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOFMT=gofmt

.PHONY: help run build test fmt fix lint clean

# Help
help: ## Show available commands
	@echo 'Available commands:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-10s %s\n", $$1, $$2}'

# Development
run: ## Run the application
	@$(GOCMD) run cmd/server/main.go

dev: ## Run with hot reload (requires air)	
	@air || $(GOCMD) run cmd/server/main.go

# Building
build: ## Build the binary
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/server/main.go

# Testing
test: ## Run tests
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Code quality
fmt: ## Format code and check formatting
	@echo "Formatting code..."
	@$(GOFMT) -s -w .
	@$(GOCMD) mod tidy
	@echo "Checking formatting..."
	@test -z "$$($(GOFMT) -s -l .)" && echo "✓ Code is properly formatted" || (echo "✗ Code formatting issues found" && exit 1)

fix: ## Fix formatting and auto-fixable linting issues
	@echo "Fixing code formatting..."
	@$(GOFMT) -s -w .
	@echo "Running goimports..."
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	@goimports -w -local golinks .
	@echo "Fixing auto-fixable linting issues..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run --fix --timeout=3m ./... || echo "Some issues may require manual fixing"
	@echo "Code formatting and auto-fixes complete!"

lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run --timeout=3m ./...

# Dependencies
deps: ## Download dependencies
	@$(GOCMD) mod download
	@$(GOCMD) mod tidy

# Docker
docker-build: ## Build Docker image
	@docker build -t $(BINARY_NAME) .

docker-run: ## Run Docker container
	@docker run -p 8080:8080 --rm $(BINARY_NAME)

# Cleanup
clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f *.db