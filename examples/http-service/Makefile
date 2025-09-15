# NotifyHub HTTP Service Makefile

.PHONY: help build run test unit-test e2e-test performance-test clean deps lint fmt vet docker-build docker-run

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build the application
build: ## Build the HTTP service binary
	@echo "Building HTTP service..."
	go build -o bin/http-service ./cmd/server.go

# Run the application
run: ## Run the HTTP service
	@echo "Starting HTTP service..."
	go run ./cmd/server.go

# Install dependencies
deps: ## Install dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run all tests
test: unit-test e2e-test ## Run all tests (unit + e2e)

# Run unit tests
unit-test: ## Run unit tests
	@echo "Running unit tests..."
	go test -v -race -timeout 30s ./test/unit/...

# Run unit tests with coverage
unit-test-coverage: ## Run unit tests with coverage
	@echo "Running unit tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./test/unit/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run E2E tests (requires running server)
e2e-test: ## Run E2E tests
	@echo "Running E2E tests..."
	@echo "Make sure the server is running on port 8081 for E2E tests"
	go test -v -timeout 60s ./test/e2e/...

# Run performance tests
performance-test: ## Run performance tests
	@echo "Running performance tests..."
	@echo "Make sure the server is running on port 8080 for performance tests"
	go test -v -timeout 300s ./test/performance/...

# Run load tests with different configurations
load-test: ## Run comprehensive load tests
	@echo "Running load tests..."
	go test -v -timeout 600s -run TestLoad ./test/performance/...

# Run stress tests
stress-test: ## Run stress tests
	@echo "Running stress tests..."
	go test -v -timeout 900s -run TestStress ./test/performance/...

# Run benchmarks
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./test/unit/...
	go test -bench=. -benchmem ./test/performance/...

# Lint the code
lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run ./...

# Format the code
fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

# Vet the code
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run development server with live reload (requires air)
dev: ## Run development server with live reload
	@echo "Starting development server with live reload..."
	air -c .air.toml

# Build Docker image
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t notifyhub-http-service:latest .

# Run Docker container
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env notifyhub-http-service:latest

# Run with Docker Compose
docker-compose-up: ## Start services with Docker Compose
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

# Stop Docker Compose
docker-compose-down: ## Stop Docker Compose services
	@echo "Stopping Docker Compose services..."
	docker-compose down

# Generate API documentation
docs: ## Generate API documentation
	@echo "Generating API documentation..."
	@echo "API Documentation:" > API.md
	@echo "==================" >> API.md
	@echo "" >> API.md
	@echo "Base URL: http://localhost:8080" >> API.md
	@echo "" >> API.md
	@echo "Endpoints:" >> API.md
	@echo "- GET  /health                        - Health check" >> API.md
	@echo "- GET  /metrics                       - Service metrics" >> API.md
	@echo "- POST /api/v1/notifications          - Send notification" >> API.md
	@echo "- POST /api/v1/notifications/bulk     - Send bulk notifications" >> API.md
	@echo "- GET  /api/v1/notifications/text     - Send text notification" >> API.md
	@echo "- GET  /api/v1/docs                   - API documentation" >> API.md
	@echo "" >> API.md
	@echo "Authentication: Bearer token in Authorization header" >> API.md

# Check code quality
quality: fmt vet lint test ## Run all quality checks (format, vet, lint, test)

# Setup development environment
setup: deps ## Setup development environment
	@echo "Setting up development environment..."
	@if [ ! -f .env ]; then cp .env.example .env; echo "Created .env file from .env.example"; fi
	@echo "Setup complete!"
	@echo "1. Edit .env file with your configuration"
	@echo "2. Run 'make run' to start the service"

# Install development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Show service status
status: ## Show service status
	@echo "Checking service status..."
	@curl -s http://localhost:8080/health | jq . || echo "Service not running or jq not installed"

# Show service metrics
metrics: ## Show service metrics
	@echo "Fetching service metrics..."
	@curl -s http://localhost:8080/metrics | jq . || echo "Service not running or jq not installed"

# Example requests
example-notification: ## Send example notification
	@echo "Sending example notification..."
	@curl -X POST http://localhost:8080/api/v1/notifications \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $(API_KEY)" \
		-d '{"title":"Test","body":"Hello World","targets":[{"type":"email","value":"test@example.com"}]}' | jq .

example-bulk: ## Send example bulk notifications
	@echo "Sending example bulk notifications..."
	@curl -X POST http://localhost:8080/api/v1/notifications/bulk \
		-H "Content-Type: application/json" \
		-H "Authorization: Bearer $(API_KEY)" \
		-d '{"notifications":[{"title":"Test 1","body":"Message 1","targets":[{"type":"email","value":"test1@example.com"}]},{"title":"Test 2","body":"Message 2","targets":[{"type":"email","value":"test2@example.com"}]}]}' | jq .

example-text: ## Send example text notification
	@echo "Sending example text notification..."
	@curl "http://localhost:8080/api/v1/notifications/text?title=Quick%20Test&body=Hello&target=text@example.com" \
		-H "Authorization: Bearer $(API_KEY)" | jq .