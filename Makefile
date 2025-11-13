.PHONY: help build test run-simple run-continuous clean fmt vet

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build all examples
	@echo "Building examples..."
	@mkdir -p bin
	@go build -o bin/simple examples/simple/main.go
	@go build -o bin/continuous examples/continuous/main.go
	@go build -o bin/chaos_context examples/chaos_context/main.go
	@echo "Build complete! Binaries in bin/"

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -gcflags=all=-l ./...

race: ## Run tests with race detector
	@echo "Running race detector..."
	@go test -race -count=100 ./...

run-simple: build ## Run simple chaos example
	@echo "Running simple chaos example..."
	@./bin/simple

run-continuous: build ## Run continuous chaos testing
	@echo "Running continuous chaos testing (Ctrl+C to stop)..."
	@./bin/continuous

fmt: ## Format code
	@go fmt ./...

vet: ## Run go vet
	@go vet ./...

clean: ## Clean build artifacts
	@rm -rf bin/
	@echo "Clean complete!"

.DEFAULT_GOAL := help
