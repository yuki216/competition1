# Fixora IT Ticketing System - Makefile

.PHONY: help build run test clean migrate seed docker-build docker-run

# Default target
help:
	@echo "Fixora IT Ticketing System"
	@echo ""
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  migrate       - Run database migrations"
	@echo "  seed          - Seed database with sample data"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run with Docker Compose"
	@echo "  dev           - Run in development mode"
	@echo "  fmt           - Format Go code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run linter"

# Variables
APP_NAME = fixora
BUILD_DIR = build
MAIN_FILE = cmd/fixora/main.go
DOCKER_IMAGE = fixora:latest

# Build targets
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Build completed: $(BUILD_DIR)/$(APP_NAME)"

# Development build
dev-build:
	@echo "Building $(APP_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	go build -race -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Development build completed: $(BUILD_DIR)/$(APP_NAME)"

# Run the application
run: build
	@echo "Starting $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

# Run in development mode
dev:
	@echo "Starting $(APP_NAME) in development mode..."
	go run $(MAIN_FILE)

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean -cache
	@echo "Clean completed"

# Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.54.2"; \
	fi

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Run database migrations
migrate: build
	@echo "Running database migrations..."
	./$(BUILD_DIR)/$(APP_NAME) -migrate

# Seed database
seed: build
	@echo "Seeding database..."
	./$(BUILD_DIR)/$(APP_NAME) -seed

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-run:
	@echo "Running with Docker Compose..."
	docker-compose up -d

docker-stop:
	@echo "Stopping Docker containers..."
	docker-compose down

docker-logs:
	@echo "Showing Docker logs..."
	docker-compose logs -f

# Development setup
setup: deps
	@echo "Setting up development environment..."
	cp .env.example .env
	@echo "Development setup completed. Please edit .env file with your configuration."

# Production build (optimized)
prod-build:
	@echo "Building $(APP_NAME) for production..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Production build completed: $(BUILD_DIR)/$(APP_NAME)"

# Install tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/golang/mock/mockgen@latest
	@echo "Tools installed"

# Generate mocks
generate-mocks:
	@echo "Generating mocks..."
	@if command -v mockgen >/dev/null 2>&1; then \
		mkdir -p test/mocks; \
		mockgen -source=application/port/outbound/repository.go -destination=test/mocks/repository_mock.go; \
		mockgen -source=application/port/outbound/ai.go -destination=test/mocks/ai_mock.go; \
	else \
		echo "mockgen not installed. Run 'make install-tools' first."; \
	fi

# Run all checks (fmt, vet, lint, test)
check: fmt vet test
	@echo "All checks completed"

# Watch for changes and rebuild (requires air)
watch:
	@echo "Watching for changes..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
	fi

# Database operations
db-reset:
	@echo "Resetting database..."
	@echo "This will delete all data. Are you sure? [y/N]"
	@read -r confirm && [ "$$confirm" = "y" ] || exit 1
	dropdb fixora 2>/dev/null || true
	createdb fixora
	$(MAKE) migrate
	$(MAKE) seed

# Create database
db-create:
	@echo "Creating database..."
	createdb fixora 2>/dev/null || echo "Database already exists"
	@echo "Database created/verified"

# Backup database
db-backup:
	@echo "Creating database backup..."
	pg_dump fixora > backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "Backup completed"

# Version information
version:
	@echo "Fixora IT Ticketing System"
	@echo "Version: $(shell git describe --tags --always 2>/dev/null || echo 'unknown')"
	@echo "Build: $(shell date +%Y-%m-%d_%H:%M:%S)"
	@echo "Go: $(shell go version)"

# Quick development setup (combines multiple commands)
init-dev: setup db-create migrate
	@echo "Development environment initialized"
	@echo "Run 'make dev' to start the application in development mode"
