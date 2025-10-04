# Makefile for NotifyHub v3.0.0

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Project info
BINARY_NAME=notifyhub
BINARY_UNIX=$(BINARY_NAME)_unix

# Directories
PKG_DIR=./pkg/...
ROOT_DIR=.

.PHONY: all build clean test coverage deps fmt lint vet check help

# Default target
all: check build

# Format code using go fmt
fmt:
	@echo "Running go fmt..."
	$(GOFMT) $(PKG_DIR)
	@echo "Code formatting complete."

# Lint code using golangci-lint (more comprehensive than golint)
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run $(PKG_DIR)
	@echo "Linting complete."

# Alternative lint using staticcheck (if golangci-lint is not preferred)
lint-static:
	@echo "Running staticcheck..."
	@which staticcheck > /dev/null || (echo "staticcheck not found. Installing..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck $(PKG_DIR)
	@echo "Static analysis complete."

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) $(PKG_DIR)
	@echo "Go vet complete."

# Comprehensive check: fmt + vet + lint
check: fmt vet lint
	@echo "All code quality checks passed!"

# Build the binary
build:
	@echo "Building..."
	$(GOBUILD) -o $(BINARY_NAME) -v $(PKG_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v $(PKG_DIR)

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic $(PKG_DIR)
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u honnef.co/go/tools/cmd/staticcheck
	@echo "Development tools installed."

# Cross compilation
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(PKG_DIR)

# Development workflow
dev: deps fmt vet lint test
	@echo "Development workflow complete!"

# CI/CD workflow
ci: deps check test coverage
	@echo "CI/CD workflow complete!"

# Show help
help:
	@echo "Available targets:"
	@echo "  fmt           - Format code using go fmt"
	@echo "  lint          - Lint code using golangci-lint"
	@echo "  lint-static   - Static analysis using staticcheck"
	@echo "  vet           - Run go vet"
	@echo "  check         - Run fmt + vet + lint"
	@echo "  build         - Build the binary"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  install-tools - Install development tools"
	@echo "  build-linux   - Cross compile for Linux"
	@echo "  dev           - Complete development workflow"
	@echo "  ci            - Complete CI/CD workflow"
	@echo "  help          - Show this help message"