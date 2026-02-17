.PHONY: help run build docker-build docker-run clean test lint

# Variables
BINARY_NAME=websocket-server
DOCKER_IMAGE=notification-srv
DOCKER_TAG=latest
GO_FILES=$(shell find . -name '*.go' -type f)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Run the WebSocket server locally
	@echo "Starting WebSocket server..."
	go run ./cmd/api

test: ## Run tests
	@echo "Running tests..."
	go test -v -cover ./...

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

.DEFAULT_GOAL := help
