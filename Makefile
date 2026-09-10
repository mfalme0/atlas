.PHONY: dev build test lint benchmark clean help

BINARY_SERVER := atlas-server
BINARY_CLI := atlas-cli
BUILD_DIR := bin

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build all binaries
	@echo "Building server..."
	@go build -o $(BUILD_DIR)/$(BINARY_SERVER) ./cmd/atlas-server/
	@echo "Building CLI..."
	@go build -o $(BUILD_DIR)/$(BINARY_CLI) ./cmd/atlas-cli/
	@echo "Build complete."

dev: ## Start development server
	@go run ./cmd/atlas-server/

test: ## Run all tests
	@go test ./...

test-race: ## Run tests with race detector
	@go test -race ./...

test-verbose: ## Run tests with verbose output
	@go test -v ./...

lint: ## Run linters
	@go vet ./...
	@echo "Lint passed."

benchmark: ## Run benchmarks
	@go test -bench=. -benchmem ./...

clean: ## Remove build artifacts
	@rm -rf $(BUILD_DIR)
	@rm -f atlas-server.exe atlas-cli.exe
	@echo "Clean complete."

docker-build: ## Build Docker images
	@docker compose build

docker-up: ## Start all services
	@docker compose up -d

docker-down: ## Stop all services
	@docker compose down

docker-logs: ## View service logs
	@docker compose logs -f

fmt: ## Format Go code
	@gofmt -s -w .

mod-tidy: ## Tidy Go modules
	@go mod tidy

check: lint test ## Run lint and tests
	@echo "All checks passed."

.DEFAULT_GOAL := help
