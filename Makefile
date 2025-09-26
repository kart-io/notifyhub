# NotifyHub v3.0 Makefile
# Author: NotifyHub Development Team
# Description: Build, test, lint, and format tools for NotifyHub v3.0 Architecture

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod
GOFMT = gofmt
GOVET = $(GOCMD) vet

# Project info
MODULE_NAME = github.com/kart-io/notifyhub
BINARY_NAME = notifyhub
BINARY_UNIX = $(BINARY_NAME)_unix

# Directories
BUILD_DIR = build
COVERAGE_DIR = coverage
DOCS_DIR = docs

# Linting tools
GOLANGCI_LINT = golangci-lint
STATICCHECK = staticcheck

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[0;33m
BLUE = \033[0;34m
NC = \033[0m # No Color

.PHONY: help
help: ## Display this help message
	@echo "$(BLUE)NotifyHub v3.0 Makefile Commands:$(NC)"
	@echo "$(YELLOW)Architecture: Unified 3-layer design with Platform abstraction$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

.PHONY: all
all: clean fmt lint test build ## Run all checks and build

.PHONY: fmt
fmt: ## Format Go code using gofmt
	@echo "$(YELLOW)Formatting Go code...$(NC)"
	@$(GOFMT) -s -w .
	@echo "$(GREEN)✓ Code formatting completed$(NC)"

.PHONY: fmt-check
fmt-check: ## Check if code is properly formatted
	@echo "$(YELLOW)Checking code formatting...$(NC)"
	@unformatted=$$($(GOFMT) -l . | grep -v vendor/); \
	if [ -n "$$unformatted" ]; then \
		echo "$(RED)✗ The following files are not properly formatted:$(NC)"; \
		echo "$$unformatted"; \
		exit 1; \
	else \
		echo "$(GREEN)✓ All Go files are properly formatted$(NC)"; \
	fi

.PHONY: lint
lint: ## Run golangci-lint on v3.0 core packages
	@echo "$(YELLOW)Running linter on v3.0 core packages...$(NC)"
	@if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		$(GOLANGCI_LINT) run ./pkg/notifyhub ./pkg/queue ./pkg/logger ./internal/platform --timeout=5m; \
		echo "$(GREEN)✓ Linting completed$(NC)"; \
	else \
		echo "$(RED)golangci-lint not found. Installing...$(NC)"; \
		$(MAKE) install-lint; \
		$(GOLANGCI_LINT) run ./pkg/notifyhub ./pkg/queue ./pkg/logger ./internal/platform --timeout=5m; \
		echo "$(GREEN)✓ Linting completed$(NC)"; \
	fi

.PHONY: lint-all
lint-all: ## Run golangci-lint on all packages (including legacy)
	@echo "$(YELLOW)Running linter on all packages...$(NC)"
	@if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		$(GOLANGCI_LINT) run --timeout=5m; \
		echo "$(GREEN)✓ Linting completed$(NC)"; \
	else \
		echo "$(RED)golangci-lint not found. Installing...$(NC)"; \
		$(MAKE) install-lint; \
		$(GOLANGCI_LINT) run --timeout=5m; \
		echo "$(GREEN)✓ Linting completed$(NC)"; \
	fi

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with autofix
	@echo "$(YELLOW)Running linter with autofix...$(NC)"
	@$(GOLANGCI_LINT) run ./pkg/notifyhub ./pkg/queue ./pkg/logger ./internal/platform --fix --timeout=5m
	@echo "$(GREEN)✓ Linting with autofix completed$(NC)"

.PHONY: vet
vet: ## Run go vet on v3.0 core packages
	@echo "$(YELLOW)Running go vet on v3.0 core packages...$(NC)"
	@$(GOVET) ./pkg/notifyhub ./pkg/queue ./pkg/logger ./internal/platform
	@echo "$(GREEN)✓ go vet completed$(NC)"

.PHONY: vet-all
vet-all: ## Run go vet on all packages (including legacy)
	@echo "$(YELLOW)Running go vet on all packages...$(NC)"
	@$(GOVET) ./...
	@echo "$(GREEN)✓ go vet completed$(NC)"

.PHONY: staticcheck
staticcheck: ## Run staticcheck
	@echo "$(YELLOW)Running staticcheck...$(NC)"
	@if command -v $(STATICCHECK) >/dev/null 2>&1; then \
		$(STATICCHECK) ./...; \
		echo "$(GREEN)✓ staticcheck completed$(NC)"; \
	else \
		echo "$(RED)staticcheck not found. Installing...$(NC)"; \
		$(MAKE) install-staticcheck; \
		$(STATICCHECK) ./...; \
		echo "$(GREEN)✓ staticcheck completed$(NC)"; \
	fi

.PHONY: test
test: ## Run tests on v3.0 core packages
	@echo "$(YELLOW)Running tests on v3.0 core packages...$(NC)"
	@$(GOTEST) -v ./pkg/notifyhub ./pkg/queue ./pkg/logger ./internal/platform
	@echo "$(GREEN)✓ Tests completed$(NC)"

.PHONY: test-all
test-all: ## Run tests on all packages (including legacy)
	@echo "$(YELLOW)Running tests on all packages...$(NC)"
	@$(GOTEST) -v ./...
	@echo "$(GREEN)✓ Tests completed$(NC)"

.PHONY: test-short
test-short: ## Run tests with short flag
	@echo "$(YELLOW)Running short tests...$(NC)"
	@$(GOTEST) -short ./...
	@echo "$(GREEN)✓ Short tests completed$(NC)"

.PHONY: test-race
test-race: ## Run tests with race detector
	@echo "$(YELLOW)Running tests with race detector...$(NC)"
	@$(GOTEST) -race ./...
	@echo "$(GREEN)✓ Race tests completed$(NC)"

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "$(YELLOW)Running tests with coverage...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(GREEN)✓ Coverage report generated: $(COVERAGE_DIR)/coverage.html$(NC)"

.PHONY: test-coverage-func
test-coverage-func: ## Show test coverage by function
	@echo "$(YELLOW)Generating coverage by function...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@$(GOTEST) -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	@$(GOCMD) tool cover -func=$(COVERAGE_DIR)/coverage.out

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(YELLOW)Running integration tests...$(NC)"
	@$(GOTEST) -v ./tests/integration/...
	@echo "$(GREEN)✓ Integration tests completed$(NC)"

.PHONY: test-validation
test-validation: ## Run validation tests (tests directory)
	@echo "$(YELLOW)Running validation tests...$(NC)"
	@$(GOTEST) -v ./tests
	@echo "$(GREEN)✓ Validation tests completed$(NC)"

.PHONY: test-all
test-all: test test-integration test-validation ## Run all tests

.PHONY: test-architecture
test-architecture: ## Run architecture validation tests
	@echo "$(YELLOW)Running architecture validation tests...$(NC)"
	@$(GOTEST) -v ./tests -run TestDualInterfaceElimination
	@$(GOTEST) -v ./tests -run TestAsyncProcessingReality
	@$(GOTEST) -v ./tests -run TestStrongTypedConfigurations
	@$(GOTEST) -v ./tests -run TestSimpleGlobalStateElimination
	@$(GOTEST) -v ./tests -run TestPlatformCapabilityNegotiation
	@$(GOTEST) -v ./tests -run TestPerformanceImprovements
	@$(GOTEST) -v ./tests -run TestBackwardCompatibility
	@echo "$(GREEN)✓ Architecture validation completed$(NC)"

.PHONY: test-performance
test-performance: ## Run performance validation tests
	@echo "$(YELLOW)Running performance validation tests...$(NC)"
	@$(GOTEST) -v ./tests -run TestPerformanceImprovements
	@echo "$(GREEN)✓ Performance validation completed$(NC)"

.PHONY: test-compatibility
test-compatibility: ## Run backward compatibility tests
	@echo "$(YELLOW)Running backward compatibility tests...$(NC)"
	@$(GOTEST) -v ./tests -run TestBackwardCompatibility
	@$(GOTEST) -v ./tests -run TestDeprecationWarnings
	@echo "$(GREEN)✓ Compatibility tests completed$(NC)"

.PHONY: bench
bench: ## Run benchmarks
	@echo "$(YELLOW)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem ./...
	@echo "$(GREEN)✓ Benchmarks completed$(NC)"

.PHONY: bench-performance
bench-performance: ## Run performance benchmarks
	@echo "$(YELLOW)Running performance benchmarks...$(NC)"
	@$(GOTEST) -bench=BenchmarkArchitecturePerformance -benchmem ./tests
	@echo "$(GREEN)✓ Performance benchmarks completed$(NC)"

.PHONY: build
build: ## Build v3.0 core packages
	@echo "$(YELLOW)Building v3.0 core packages...$(NC)"
	@$(GOBUILD) -v ./pkg/notifyhub ./pkg/queue ./pkg/logger ./internal/platform
	@echo "$(GREEN)✓ Build completed$(NC)"

.PHONY: build-all
build-all: ## Build all packages (including legacy)
	@echo "$(YELLOW)Building all packages...$(NC)"
	@$(GOBUILD) -v ./...
	@echo "$(GREEN)✓ Build completed$(NC)"

.PHONY: build-examples
build-examples: ## Build example applications (if any)
	@echo "$(YELLOW)Building examples...$(NC)"
	@if [ -d "examples" ]; then \
		for example in examples/*/; do \
			if [ -f "$$example/main.go" ]; then \
				echo "Building $$example..."; \
				cd "$$example" && $(GOBUILD) -v .; \
				cd ../..; \
			fi; \
		done; \
		echo "$(GREEN)✓ Examples build completed$(NC)"; \
	else \
		echo "$(BLUE)No examples directory found$(NC)"; \
	fi

.PHONY: build-check
build-check: ## Check that all packages build successfully
	@echo "$(YELLOW)Checking build for all packages...$(NC)"
	@$(GOBUILD) -v ./...
	@echo "$(GREEN)✓ All packages build successfully$(NC)"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@echo "$(GREEN)✓ Clean completed$(NC)"

.PHONY: deps
deps: ## Download dependencies
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	@$(GOMOD) download
	@echo "$(GREEN)✓ Dependencies downloaded$(NC)"

.PHONY: deps-tidy
deps-tidy: ## Tidy dependencies
	@echo "$(YELLOW)Tidying dependencies...$(NC)"
	@$(GOMOD) tidy
	@echo "$(GREEN)✓ Dependencies tidied$(NC)"

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	@echo "$(YELLOW)Verifying dependencies...$(NC)"
	@$(GOMOD) verify
	@echo "$(GREEN)✓ Dependencies verified$(NC)"

.PHONY: deps-graph
deps-graph: ## Show dependency graph
	@echo "$(YELLOW)Generating dependency graph...$(NC)"
	@$(GOCMD) mod graph

.PHONY: run-examples
run-examples: ## Run example applications (if any)
	@echo "$(YELLOW)Running examples...$(NC)"
	@if [ -d "examples" ]; then \
		for example in examples/*/; do \
			if [ -f "$$example/main.go" ]; then \
				echo "Running $$example..."; \
				cd "$$example" && $(GOCMD) run . && cd ../..; \
			fi; \
		done; \
	else \
		echo "$(BLUE)No examples directory found. This is a library project.$(NC)"; \
	fi

.PHONY: install-tools
install-tools: install-lint install-staticcheck ## Install development tools

.PHONY: install-lint
install-lint: ## Install golangci-lint
	@echo "$(YELLOW)Installing golangci-lint...$(NC)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2
	@echo "$(GREEN)✓ golangci-lint installed$(NC)"

.PHONY: install-staticcheck
install-staticcheck: ## Install staticcheck
	@echo "$(YELLOW)Installing staticcheck...$(NC)"
	@$(GOGET) honnef.co/go/tools/cmd/staticcheck
	@echo "$(GREEN)✓ staticcheck installed$(NC)"

.PHONY: security
security: ## Run security checks
	@echo "$(YELLOW)Running security checks...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "$(RED)gosec not found. Installing...$(NC)"; \
		$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec; \
		gosec ./...; \
	fi
	@echo "$(GREEN)✓ Security checks completed$(NC)"

.PHONY: docs
docs: ## Generate documentation
	@echo "$(YELLOW)Generating documentation...$(NC)"
	@mkdir -p $(DOCS_DIR)
	@$(GOCMD) doc -all ./... > $(DOCS_DIR)/api.txt
	@echo "$(GREEN)✓ Documentation generated: $(DOCS_DIR)/api.txt$(NC)"

.PHONY: godoc
godoc: ## Start godoc server
	@echo "$(YELLOW)Starting godoc server...$(NC)"
	@echo "$(BLUE)Visit: http://localhost:6060/pkg/$(MODULE_NAME)/$(NC)"
	@godoc -http=:6060

.PHONY: version
version: ## Show Go version and environment info
	@echo "$(BLUE)Go version and environment:$(NC)"
	@$(GOCMD) version
	@echo ""
	@$(GOCMD) env

.PHONY: check
check: fmt-check vet lint ## Run all checks without fixing on refactored packages

.PHONY: check-all
check-all: fmt-check vet-all lint-all staticcheck ## Run all checks without fixing on all packages

.PHONY: pre-commit
pre-commit: fmt lint test ## Run pre-commit checks

.PHONY: ci
ci: check test-coverage ## Run CI pipeline

.PHONY: release-check
release-check: clean fmt-check lint staticcheck test-all build-all ## Complete release check

.PHONY: v3-check
v3-check: fmt-check vet lint test test-architecture test-performance test-compatibility ## Complete v3.0 validation check

.PHONY: v3-quick
v3-quick: fmt test ## Quick v3.0 development check

.PHONY: migration-test
migration-test: ## Test migration scenarios
	@echo "$(YELLOW)Testing migration scenarios...$(NC)"
	@$(GOTEST) -v ./tests -run TestBackwardCompatibility
	@$(GOTEST) -v ./tests -run TestDeprecationWarnings
	@echo "$(GREEN)✓ Migration tests completed$(NC)"

.PHONY: docs-generate
docs-generate: ## Generate v3.0 documentation
	@echo "$(YELLOW)Generating v3.0 documentation...$(NC)"
	@mkdir -p $(DOCS_DIR)/v3
	@$(GOCMD) doc -all ./pkg/notifyhub > $(DOCS_DIR)/v3/notifyhub-api.txt
	@echo "  Platforms are now embedded in individual platform packages"
	@$(GOCMD) doc -all ./pkg/queue > $(DOCS_DIR)/v3/queue-api.txt
	@$(GOCMD) doc -all ./pkg/logger > $(DOCS_DIR)/v3/logger-api.txt
	@echo "$(GREEN)✓ v3.0 documentation generated: $(DOCS_DIR)/v3/$(NC)"

.PHONY: show-v3-status
show-v3-status: ## Show v3.0 architecture status
	@echo "$(BLUE)NotifyHub v3.0 Architecture Status:$(NC)"
	@echo "$(GREEN)✓ Unified Platform interface$(NC)"
	@echo "$(GREEN)✓ Strong-typed configuration$(NC)"
	@echo "$(GREEN)✓ True async processing$(NC)"
	@echo "$(GREEN)✓ Global state elimination$(NC)"
	@echo "$(GREEN)✓ Middleware system$(NC)"
	@echo "$(GREEN)✓ Backward compatibility$(NC)"
	@echo ""
	@echo "$(YELLOW)Core packages:$(NC)"
	@echo "  - pkg/notifyhub/     # Main Hub implementation"
	@echo "  - pkg/platforms/     # Platform adapters (integrated into main hub)"
	@echo "  - pkg/queue/         # Queue system"
	@echo "  - pkg/logger/        # Logging interface"
	@echo "  - internal/platform/ # Platform internals"
	@echo ""
	@echo "$(YELLOW)Test suites:$(NC)"
	@echo "  - tests/             # Architecture validation"
	@echo "  - examples/          # Usage examples"

# Default target
.DEFAULT_GOAL := help