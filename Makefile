.PHONY: help run test lint deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Run the notification service
	@echo "Generating swagger"
	@swag init -g cmd/server/main.go --parseVendor
	@sed -i '' '/LeftDelim:/d' docs/docs.go
	@sed -i '' '/RightDelim:/d' docs/docs.go
	@echo "Running the application"
	@go run cmd/server/main.go

test: 
	@echo "Running tests..."
	@go test -mod=vendor -coverprofile=coverage.out -failfast -timeout 5m ./internal/...
	@grep -v 'mock_' coverage.out > c.out
	@go tool cover -func=c.out | grep '^total:'
	@echo "Coverage summary generated"
	@rm -f *.out

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

.DEFAULT_GOAL := help
