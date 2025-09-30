# conx CMDB Development Makefile
# This Makefile provides convenient targets for development tasks

.PHONY: help setup start stop restart logs status test test-unit test-integration build clean reset fmt lint vet docker-build docker-run docker-test

# Default target
help:
	@echo "conx CMDB Development Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  setup           Set up the development environment"
	@echo "  start           Start all services"
	@echo "  stop            Stop all services"
	@echo "  restart         Restart all services"
	@echo "  logs            Show logs for all services"
	@echo "  logs-api        Show logs for API service"
	@echo "  logs-frontend   Show logs for frontend service"
	@echo "  logs-db         Show logs for database services"
	@echo "  status          Show status of all services"
	@echo "  test            Run all tests"
	@echo "  test-unit       Run unit tests only"
	@echo "  test-integration Run integration tests only"
	@echo "  test-coverage   Run tests with coverage report"
	@echo "  build           Build the application"
	@echo "  build-api       Build API binary"
	@echo "  build-frontend  Build frontend"
	@echo "  clean           Clean up containers and volumes"
	@echo "  reset           Reset the entire development environment"
	@echo "  fmt             Format Go code"
	@echo "  lint            Run linter on Go code"
	@echo "  vet             Run vet on Go code"
	@echo "  deps            Download Go dependencies"
	@echo "  deps-update     Update Go dependencies"
	@echo "  docker-build    Build Docker images"
	@echo "  docker-run      Run application in Docker"
	@echo "  docker-test     Run tests in Docker"
	@echo ""
	@echo "Examples:"
	@echo "  make setup      # Set up development environment"
	@echo "  make start      # Start all services"
	@echo "  make test-unit  # Run unit tests only"

# Development environment setup
setup:
	@echo "Setting up development environment..."
	@chmod +x scripts/dev.sh
	@./scripts/dev.sh setup

# Service management
start:
	@echo "Starting all services..."
	@./scripts/dev.sh start

stop:
	@echo "Stopping all services..."
	@./scripts/dev.sh stop

restart:
	@echo "Restarting all services..."
	@./scripts/dev.sh restart

# Logs
logs:
	@echo "Showing logs for all services..."
	@./scripts/dev.sh logs

logs-api:
	@echo "Showing logs for API service..."
	@./scripts/dev.sh logs api

logs-frontend:
	@echo "Showing logs for frontend service..."
	@./scripts/dev.sh logs frontend

logs-db:
	@echo "Showing logs for database services..."
	@./scripts/dev.sh logs postgres

# Status
status:
	@echo "Showing service status..."
	@./scripts/dev.sh status

# Testing
test:
	@echo "Running all tests..."
	@./scripts/dev.sh test

test-unit:
	@echo "Running unit tests..."
	@./scripts/dev.sh test unit

test-integration:
	@echo "Running integration tests..."
	@./scripts/dev.sh test int

test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Building
build:
	@echo "Building application..."
	@./scripts/dev.sh build

build-api:
	@echo "Building API binary..."
	@mkdir -p bin
	@go build -o bin/api ./cmd/api

build-frontend:
	@echo "Building frontend..."
	@if [ -d "web" ]; then \
		cd web && \
		npm install && \
		npm run build; \
	else \
		echo "Frontend directory not found. Skipping frontend build."; \
	fi

# Cleanup
clean:
	@echo "Cleaning up containers and volumes..."
	@./scripts/dev.sh clean

reset:
	@echo "Resetting development environment..."
	@./scripts/dev.sh reset

# Go code quality
fmt:
	@echo "Formatting Go code..."
	@go fmt ./...

lint:
	@echo "Running linter on Go code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run ./...; \
	fi

vet:
	@echo "Running vet on Go code..."
	@go vet ./...

# Dependencies
deps:
	@echo "Downloading Go dependencies..."
	@go mod download
	@go mod tidy

deps-update:
	@echo "Updating Go dependencies..."
	@go get -u ./...
	@go mod tidy

# Docker operations
docker-build:
	@echo "Building Docker images..."
	@docker-compose build

docker-run:
	@echo "Running application in Docker..."
	@docker-compose up

docker-test:
	@echo "Running tests in Docker..."
	@docker-compose -f docker-compose.yml -f docker-compose.test.yml run --rm test

# Development workflow
dev-setup: setup deps
	@echo "Development environment setup complete!"

dev-start: start
	@echo "Services started! Access them at:"
	@echo "  API: http://localhost:8080"
	@echo "  Frontend: http://localhost:3000"
	@echo "  Neo4j Browser: http://localhost:7474"

dev-test: test-unit fmt lint vet
	@echo "Development tests complete!"

dev-build: build-api build-frontend
	@echo "Application build complete!"

# Quick development cycle
dev: fmt vet test-unit build-api
	@echo "Quick development cycle complete!"

# Production build
prod-build:
	@echo "Building for production..."
	@CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api ./cmd/api
	@if [ -d "web" ]; then \
		cd web && \
		NODE_ENV=production npm install && \
		npm run build; \
	fi

# Database operations
db-migrate:
	@echo "Running database migrations..."
	@if command -v goose >/dev/null 2>&1; then \
		goose -dir=migrations postgres "postgres://cmdb_user:dev_password@localhost:5432/cmdb?sslmode=disable" up; \
	else \
		echo "goose not found. Installing..."; \
		go install github.com/pressly/goose/v3/cmd/goose@latest; \
		goose -dir=migrations postgres "postgres://cmdb_user:dev_password@localhost:5432/cmdb?sslmode=disable" up; \
	fi

db-rollback:
	@echo "Rolling back database migrations..."
	@if command -v goose >/dev/null 2>&1; then \
		goose -dir=migrations postgres "postgres://cmdb_user:dev_password@localhost:5432/cmdb?sslmode=disable" down; \
	else \
		echo "goose not found. Installing..."; \
		go install github.com/pressly/goose/v3/cmd/goose@latest; \
		goose -dir=migrations postgres "postgres://cmdb_user:dev_password@localhost:5432/cmdb?sslmode=disable" down; \
	fi

# Security checks
security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Installing..."; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		gosec ./...; \
	fi

# Performance testing
bench:
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem ./...

# Generate documentation
docs:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Documentation available at: http://localhost:6060/pkg/connect/"; \
		godoc -http=:6060; \
	else \
		echo "godoc not found. Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/pressly/goose/v3/cmd/goose@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/tools/cmd/godoc@latest
	@go install github.com/stretchr/testify/mock/mockgen@latest
	@echo "Development tools installed!"

# Check if all required tools are installed
check-tools:
	@echo "Checking required tools..."
	@command -v docker >/dev/null 2>&1 || (echo "Docker is not installed" && exit 1)
	@command -v docker-compose >/dev/null 2>&1 || (echo "Docker Compose is not installed" && exit 1)
	@command -v go >/dev/null 2>&1 || (echo "Go is not installed" && exit 1)
	@command -v node >/dev/null 2>&1 || (echo "Node.js is not installed" && exit 1)
	@command -v npm >/dev/null 2>&1 || (echo "npm is not installed" && exit 1)
	@echo "All required tools are installed!"

# Pre-commit hooks
pre-commit: fmt vet lint test-unit
	@echo "Pre-commit checks passed!"

# Quick check before pushing
pre-push: test fmt lint vet security
	@echo "Pre-push checks passed!"

# Create release (simplified)
release: clean prod-build
	@echo "Release build complete!"
	@echo "Binaries are available in the bin/ directory"

# Watch mode for development
watch:
	@echo "Starting development watch mode..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not found. Installing..."; \
		go install github.com/cosmtrek/air@latest; \
		air; \
	fi

# Initialize project for new developers
init: check-tools install-tools deps dev-setup
	@echo "Project initialized successfully!"
	@echo "Run 'make start' to begin development."
