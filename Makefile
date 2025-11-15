include .env
export
BINARY=engine

models:
	@echo "Generating models"
	@sqlboiler psql

swagger:
	@echo "Generating swagger"
	@swag init -g cmd/api/main.go
	@echo "Fixing swagger docs (removing deprecated LeftDelim/RightDelim)..."
	@sed -i '' '/LeftDelim:/d' docs/docs.go
	@sed -i '' '/RightDelim:/d' docs/docs.go

run-api:
	@echo "Generating swagger"
	@swag init -g cmd/api/main.go
	@sed -i '' '/LeftDelim:/d' docs/docs.go
	@sed -i '' '/RightDelim:/d' docs/docs.go
	@echo "Running the application"
	@go run cmd/api/main.go

run-consumer:
	@echo "Running the consumer"
	@go run cmd/consumer/main.go

build-docker-compose:
	@echo "make models first"
	@make models
	@echo "Building docker compose"
	docker compose up --build -d