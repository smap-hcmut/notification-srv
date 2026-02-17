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

build: ## Build the binary
	@echo "Building binary..."
	go build -o $(BINARY_NAME) -ldflags="-s -w" ./cmd/api
	@echo "Binary built: $(BINARY_NAME)"

docker-build: ## Build Docker image (local platform)
	@echo "Building Docker image for local platform..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f cmd/api/Dockerfile .

docker-build-amd64: ## Build Docker image for AMD64 servers
	@echo "Building Docker image for AMD64..."
	docker buildx build --platform linux/amd64 -t $(DOCKER_IMAGE):$(DOCKER_TAG) -f cmd/api/Dockerfile .

docker-run: docker-build ## Build and run Docker container
	@echo "Running Docker container..."
	docker run -p 8081:8081 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	docker rmi -f $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true

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
