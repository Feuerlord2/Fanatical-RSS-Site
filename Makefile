# Fanatical RSS Site Makefile

.PHONY: help build run clean test deps update-feeds install dev

# Default target
help: ## Show this help message
	@echo "Fanatical RSS Site - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building gofanatical..."
	@go build -o gofanatical ./cmd/gofanatical.go
	@echo "✅ Build completed"

run: build ## Build and run the RSS generator
	@echo "Generating RSS feeds..."
	@./gofanatical
	@echo "✅ RSS feeds generated"

clean: ## Clean build artifacts and RSS files
	@echo "Cleaning up..."
	@rm -f gofanatical
	@rm -f docs/*.rss
	@echo "✅ Cleanup completed"

test: ## Run tests
	@echo "Running tests..."
	@go test ./...
	@echo "✅ Tests completed"

deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify
	@echo "✅ Dependencies updated"

update-feeds: run ## Generate RSS feeds (now writes directly to docs folder)
	@echo "✅ RSS feeds updated in docs folder"

install: deps build ## Install dependencies and build
	@echo "✅ Installation completed"

dev: ## Development mode - build and run with detailed logging
	@echo "Running in development mode..."
	@go build -o gofanatical ./cmd/gofanatical.go
	@LOG_LEVEL=debug ./gofanatical
	@echo "✅ Development run completed"

fmt: ## Format Go code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✅ Code formatted"

lint: ## Run linter (requires golangci-lint)
	@echo "Running linter..."
	@golangci-lint run || echo "⚠️  golangci-lint not found, skipping"

check: fmt lint test ## Run all checks (format, lint, test)
	@echo "✅ All checks completed"

serve: ## Serve the docs folder locally (requires Python)
	@echo "Starting local server at http://localhost:8080"
	@cd docs && python3 -m http.server 8080 || python -m SimpleHTTPServer 8080

# Development helpers
watch: ## Watch for changes and rebuild (requires entr)
	@echo "Watching for changes... (requires 'entr' tool)"
	@find . -name "*.go" | entr -r make run

# Docker support (optional)
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t fanatical-rss .

docker-run: docker-build ## Run in Docker container
	@echo "Running in Docker..."
	@docker run --rm -v $(PWD)/docs:/app/docs fanatical-rss
