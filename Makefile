.PHONY: build run test test-unit test-integration migrate-up migrate-down clean

# Build the application
build:
	go build -o bin/auth-service cmd/server/main.go

# Run the application
run:
	go run cmd/server/main.go

# Run all tests
test:
	go test -v ./...

# Run unit tests only
test-unit:
	go test -v ./test/unit/...

# Run integration tests only
test-integration:
	go test -v ./test/integration/...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Database migrations
migrate-up:
	@echo "Running database migrations..."
	@psql $(DATABASE_URL) -f migrations/001_create_auth_tables.sql

migrate-down:
	@echo "Rolling back database migrations..."
	@psql $(DATABASE_URL) -c "DROP TABLE IF EXISTS refresh_tokens; DROP TABLE IF EXISTS users;"

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	@go mod download
	@cp .env.example .env
	@echo "Please update .env file with your database configuration"

# Lint the code
lint:
	golangci-lint run

# Format the code
fmt:
	go fmt ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run the application with hot reload (requires air)
dev:
	air

# Docker commands
docker-build:
	docker build -t auth-service .

docker-run:
	docker run -p 8080:8080 --env-file .env auth-service