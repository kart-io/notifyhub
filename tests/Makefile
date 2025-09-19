# Makefile for NotifyHub Tests

# Variables
TEST_DIR := tests
COVERAGE_DIR := coverage
TIMEOUT_UNIT := 10m
TIMEOUT_INTEGRATION := 15m
TIMEOUT_PERFORMANCE := 30m
VERBOSE :=

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m

# Default target
.PHONY: help
help:
	@echo "NotifyHub Test Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make test              - Run all tests"
	@echo "  make test-unit         - Run unit tests"
	@echo "  make test-integration  - Run integration tests"
	@echo "  make test-performance  - Run performance tests"
	@echo "  make test-benchmark    - Run benchmark tests"
	@echo "  make test-coverage     - Generate test coverage report"
	@echo "  make test-race         - Run tests with race detector"
	@echo "  make test-verbose      - Run tests with verbose output"
	@echo "  make test-clean        - Clean test cache and temporary files"
	@echo ""
	@echo "Advanced:"
	@echo "  make test-ci           - Run tests in CI mode"
	@echo "  make test-watch        - Watch files and run tests on change"
	@echo "  make test-profile      - Run tests with CPU/memory profiling"

# Run all tests
.PHONY: test
test:
	@echo "$(BLUE)[INFO]$(NC) Running all tests..."
	@go test $(VERBOSE) -timeout=$(TIMEOUT_INTEGRATION) ./$(TEST_DIR)/...
	@echo "$(GREEN)[SUCCESS]$(NC) All tests passed"

# Run unit tests
.PHONY: test-unit
test-unit:
	@echo "$(BLUE)[INFO]$(NC) Running unit tests..."
	@go test $(VERBOSE) -timeout=$(TIMEOUT_UNIT) ./$(TEST_DIR)/unit/...
	@echo "$(GREEN)[SUCCESS]$(NC) Unit tests passed"

# Run integration tests
.PHONY: test-integration
test-integration:
	@echo "$(BLUE)[INFO]$(NC) Running integration tests..."
	@go test $(VERBOSE) -timeout=$(TIMEOUT_INTEGRATION) ./$(TEST_DIR)/integration/...
	@echo "$(GREEN)[SUCCESS]$(NC) Integration tests passed"

# Run performance tests
.PHONY: test-performance
test-performance:
	@echo "$(BLUE)[INFO]$(NC) Running performance tests..."
	@go test $(VERBOSE) -timeout=$(TIMEOUT_PERFORMANCE) ./$(TEST_DIR)/performance/...
	@echo "$(GREEN)[SUCCESS]$(NC) Performance tests passed"

# Run benchmark tests
.PHONY: test-benchmark
test-benchmark:
	@echo "$(BLUE)[INFO]$(NC) Running benchmark tests..."
	@go test -bench=. -benchmem -benchtime=10s ./$(TEST_DIR)/performance/...
	@echo "$(GREEN)[SUCCESS]$(NC) Benchmark tests completed"

# Generate test coverage
.PHONY: test-coverage
test-coverage:
	@echo "$(BLUE)[INFO]$(NC) Generating test coverage..."
	@mkdir -p $(COVERAGE_DIR)
	@go test -coverprofile=$(COVERAGE_DIR)/unit.out ./$(TEST_DIR)/unit/...
	@go test -coverprofile=$(COVERAGE_DIR)/integration.out ./$(TEST_DIR)/integration/...
	@echo "mode: set" > $(COVERAGE_DIR)/coverage.out
	@tail -q -n +2 $(COVERAGE_DIR)/*.out >> $(COVERAGE_DIR)/coverage.out
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "$(GREEN)[SUCCESS]$(NC) Coverage report generated: $(COVERAGE_DIR)/coverage.html"

# Run tests with race detector
.PHONY: test-race
test-race:
	@echo "$(BLUE)[INFO]$(NC) Running tests with race detector..."
	@go test -race -timeout=$(TIMEOUT_INTEGRATION) ./$(TEST_DIR)/...
	@echo "$(GREEN)[SUCCESS]$(NC) No race conditions detected"

# Run tests with verbose output
.PHONY: test-verbose
test-verbose:
	@$(MAKE) VERBOSE="-v" test

# Clean test cache
.PHONY: test-clean
test-clean:
	@echo "$(BLUE)[INFO]$(NC) Cleaning test cache..."
	@go clean -testcache
	@rm -rf $(COVERAGE_DIR)
	@find . -name "*.test" -type f -delete
	@find . -name "*.out" -type f -delete
	@echo "$(GREEN)[SUCCESS]$(NC) Test cache cleaned"

# Run tests in CI mode
.PHONY: test-ci
test-ci:
	@echo "$(BLUE)[INFO]$(NC) Running tests in CI mode..."
	@go test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -timeout=$(TIMEOUT_INTEGRATION) ./$(TEST_DIR)/...
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "$(GREEN)[SUCCESS]$(NC) CI tests completed"

# Watch files and run tests on change (requires entr)
.PHONY: test-watch
test-watch:
	@echo "$(BLUE)[INFO]$(NC) Watching for file changes..."
	@if command -v entr > /dev/null; then \
		find . -name "*.go" | entr -c make test; \
	else \
		echo "$(RED)[ERROR]$(NC) entr not installed. Install with: brew install entr (macOS) or apt-get install entr (Linux)"; \
		exit 1; \
	fi

# Run tests with profiling
.PHONY: test-profile
test-profile:
	@echo "$(BLUE)[INFO]$(NC) Running tests with profiling..."
	@mkdir -p $(COVERAGE_DIR)/profiles
	@go test -cpuprofile=$(COVERAGE_DIR)/profiles/cpu.prof -memprofile=$(COVERAGE_DIR)/profiles/mem.prof -bench=. ./$(TEST_DIR)/performance/...
	@echo "$(GREEN)[SUCCESS]$(NC) Profiling data saved to $(COVERAGE_DIR)/profiles/"
	@echo "View CPU profile: go tool pprof $(COVERAGE_DIR)/profiles/cpu.prof"
	@echo "View Memory profile: go tool pprof $(COVERAGE_DIR)/profiles/mem.prof"

# Run specific test by name
.PHONY: test-run
test-run:
	@if [ -z "$(TEST_NAME)" ]; then \
		echo "$(RED)[ERROR]$(NC) TEST_NAME not specified. Usage: make test-run TEST_NAME=TestHubCreation"; \
		exit 1; \
	fi
	@echo "$(BLUE)[INFO]$(NC) Running test: $(TEST_NAME)"
	@go test -v -run=$(TEST_NAME) ./$(TEST_DIR)/...

# Quick test (unit tests only, no coverage)
.PHONY: test-quick
test-quick:
	@echo "$(BLUE)[INFO]$(NC) Running quick tests..."
	@go test -short ./$(TEST_DIR)/unit/...
	@echo "$(GREEN)[SUCCESS]$(NC) Quick tests passed"

# Lint test code
.PHONY: test-lint
test-lint:
	@echo "$(BLUE)[INFO]$(NC) Linting test code..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./$(TEST_DIR)/...; \
	else \
		echo "$(YELLOW)[WARNING]$(NC) golangci-lint not installed. Install from: https://golangci-lint.run/usage/install/"; \
		go vet ./$(TEST_DIR)/...; \
	fi
	@echo "$(GREEN)[SUCCESS]$(NC) Test code linted"

# Generate test report
.PHONY: test-report
test-report: test-coverage
	@echo "$(BLUE)[INFO]$(NC) Generating test report..."
	@mkdir -p $(COVERAGE_DIR)/reports
	@go test -v -json ./$(TEST_DIR)/... > $(COVERAGE_DIR)/reports/test-results.json 2>&1 || true
	@echo "$(GREEN)[SUCCESS]$(NC) Test report generated: $(COVERAGE_DIR)/reports/test-results.json"

# Default verbose flag for debugging
ifeq ($(DEBUG),true)
	VERBOSE := -v
endif

.DEFAULT_GOAL := help