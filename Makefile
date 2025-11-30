.PHONY: build run clean test help

# Build configuration
BINARY_NAME=organizer
CMD_PATH=./cmd/organizer
BIN_DIR=./bin

help: ## Display this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the application binary
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Binary created at $(BIN_DIR)/$(BINARY_NAME)"

run: ## Run the application from source
	@go run $(CMD_PATH)

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete"

test: ## Run tests
	@go test -v ./...

fmt: ## Format code
	@go fmt ./...

vet: ## Run go vet
	@go vet ./...

lint: fmt vet ## Run linters (fmt and vet)

deps: ## Download dependencies
	@go mod download
	@go mod tidy

all: clean lint test build ## Run clean, lint, test, and build

