# Fanatical RSS Site Makefile

.PHONY: help build run clean test deps fmt lint check serve watch

# Default target
help: ## Show this help message
	@echo "Fanatical RSS Site - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building gofanatical..."
	@go build -o gofanatical ./cmd/
	@echo "✅ Build completed"

run: build ## Build and run the RSS generator (writes to docs/)
	@echo "Generating RSS feeds..."
	@./gofanatical
	@echo "✅ RSS feeds generated"

dev: build ## Build and run with debug logging
	@LOG_LEVEL=debug ./gofanatical

clean: ## Clean build artifacts and RSS files
	@echo "Cleaning up..."
	@rm -f gofanatical
	@rm -f docs/*.rss
	@echo "✅ Cleanup completed"

test: ## Run tests
	@go test ./...

deps: ## Download and verify dependencies
	@go mod download
	@go mod verify
	@echo "✅ Dependencies verified"

fmt: ## Format Go code
	@go fmt ./...

lint: ## Run linter (requires golangci-lint)
	@golangci-lint run || echo "⚠️  golangci-lint not found, skipping"

check: fmt lint test ## Run all checks (format, lint, test)
	@echo "✅ All checks completed"

serve: ## Serve the docs folder locally
	@echo "Starting local server at http://localhost:8080"
	@cd docs && python3 -m http.server 8080

watch: ## Watch for changes and rebuild (requires entr)
	@find . -name "*.go" | entr -r make run
