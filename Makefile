# GoLinks Makefile

BINARY_NAME=golinks
BUILD_DIR=./build
FRONTEND_DIR=./web/frontend

GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOFMT=gofmt

.PHONY: help run build dev test fmt fix lint deps clean \
        frontend-install frontend-build frontend-dev \
        docker-build docker-run ci

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# --- Frontend ---------------------------------------------------------------

frontend-install: ## Install frontend dependencies
	@cd $(FRONTEND_DIR) && npm ci || (cd $(FRONTEND_DIR) && npm install)

frontend-build: ## Build the Vite/React SPA into web/frontend/dist
	@cd $(FRONTEND_DIR) && npm run build

frontend-dev: ## Run the Vite dev server (proxies /api and /query to :8080)
	@cd $(FRONTEND_DIR) && npm run dev

# --- Go ---------------------------------------------------------------------

run: frontend-build ## Run the application (builds frontend first)
	@$(GOCMD) run ./cmd/server

dev: ## Run Go (air if available) and Vite dev server concurrently
	@command -v air >/dev/null 2>&1 && \
	  (trap 'kill 0' INT TERM; air & (cd $(FRONTEND_DIR) && npm run dev); wait) || \
	  (trap 'kill 0' INT TERM; $(GOCMD) run ./cmd/server & (cd $(FRONTEND_DIR) && npm run dev); wait)

build: frontend-build ## Build the single-binary production artifact
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

test: ## Run Go tests
	@$(GOTEST) -v -race ./...

fmt: ## Format Go code and check formatting
	@$(GOFMT) -s -w .
	@$(GOCMD) mod tidy
	@test -z "$$($(GOFMT) -s -l .)" && echo "✓ Code is properly formatted" || (echo "✗ Code formatting issues found" && exit 1)

fix: ## Auto-fix Go formatting and linting
	@$(GOFMT) -s -w .
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	@goimports -w -local golinks .
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run --fix --timeout=3m ./... || echo "Some issues may require manual fixing"

lint: ## Run Go linter
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run --timeout=3m ./...

deps: ## Download Go dependencies
	@$(GOCMD) mod download
	@$(GOCMD) mod tidy

ci: frontend-install frontend-build lint test build ## Run the full CI pipeline

# --- Docker -----------------------------------------------------------------

docker-build: ## Build Docker image
	@docker build -t $(BINARY_NAME) .

docker-run: ## Run Docker container
	@docker run -p 8080:8080 --rm $(BINARY_NAME)

# --- Cleanup ----------------------------------------------------------------

clean: ## Clean build artifacts
	@rm -rf $(BUILD_DIR) $(FRONTEND_DIR)/dist
	@rm -f *.db
