# Test Configuration

This directory contains all test suites for the NotifyHub project.

## Directory Structure

```
tests/
├── unit/               # Unit tests for individual components
├── integration/        # Integration tests for system workflows
├── performance/        # Performance and benchmark tests
├── mocks/              # Mock implementations for testing
├── fixtures/           # Test data and fixtures
├── utils/              # Test utilities and helpers
├── run_tests.sh        # Main test runner script
├── test.env            # Test environment configuration
├── Makefile            # Make targets for testing
└── README.md           # This file
```

## Running Tests

### Using the Test Runner Script

```bash
# Run all tests
./run_tests.sh all

# Run specific test suites
./run_tests.sh unit          # Run unit tests
./run_tests.sh integration   # Run integration tests
./run_tests.sh performance   # Run performance tests
./run_tests.sh benchmark     # Run benchmark tests

# With options
./run_tests.sh unit -v                    # Verbose output
./run_tests.sh integration -f TestE2E     # Filter specific tests
./run_tests.sh benchmark -f Queue         # Run specific benchmarks

# Generate coverage report
./run_tests.sh coverage
```

### Using Make

```bash
# Run all tests
make test

# Run specific test suites
make test-unit
make test-integration
make test-performance
make test-benchmark

# Generate coverage
make test-coverage

# Clean test cache
make test-clean
```

### Using Go Commands Directly

```bash
# Run all tests
go test ./tests/...

# Run with verbose output
go test -v ./tests/...

# Run specific package tests
go test ./tests/unit
go test ./tests/integration
go test ./tests/performance

# Run specific test
go test -run TestHubCreation ./tests/unit

# Run benchmarks
go test -bench=. ./tests/performance

# Generate coverage
go test -cover ./tests/...
go test -coverprofile=coverage.out ./tests/...
go tool cover -html=coverage.out
```

## Test Categories

### Unit Tests (`tests/unit/`)

Tests individual components in isolation:
- Message creation and validation
- Hub lifecycle management
- Transport registration
- Middleware chain execution
- Target validation

### Integration Tests (`tests/integration/`)

Tests complete workflows and system integration:
- End-to-end notification flow
- Multi-platform message delivery
- Routing engine integration
- Queue processing
- Error handling and recovery

### Performance Tests (`tests/performance/`)

Tests system performance and resource usage:
- Message throughput benchmarks
- Concurrent operation tests
- Memory allocation analysis
- Latency measurements
- Scalability tests
- Stress testing

## Writing Tests

### Test Utilities

Use the provided test helpers for consistent assertions:

```go
func TestExample(t *testing.T) {
    helper := utils.NewTestHelper(t)

    // Assertions
    helper.AssertEqual(expected, actual, "Values should match")
    helper.AssertNoError(err, "Should not error")
    helper.AssertTrue(condition, "Condition should be true")

    // Create test objects
    msg := utils.CreateTestMessage("Title", "Body", 3)
    target := utils.CreateTestTarget(sending.TargetTypeEmail, "test@example.com", "email")
}
```

### Using Mocks

```go
// Create mock transport
transport := mocks.NewMockTransport("test")
transport.SetDelay(100 * time.Millisecond)
transport.SetError("test@example.com", errors.New("mock error"))

// Create mock logger
logger := mocks.NewMockLogger()

// Verify interactions
calls := transport.GetCalls()
helper.AssertEqual(1, len(calls), "Should have 1 call")

messages := logger.GetMessages()
helper.AssertTrue(logger.HasError(), "Should have error logs")
```

## Configuration

Test behavior can be configured via environment variables in `test.env`:

```bash
# Set test environment
export TEST_ENV=development

# Configure timeouts
export TEST_TIMEOUT_UNIT=10m
export TEST_TIMEOUT_INTEGRATION=15m

# Enable verbose output
export VERBOSE=true

# Set performance test parameters
export PERF_NUM_MESSAGES=10000
export PERF_NUM_GOROUTINES=100
```

## Continuous Integration

The test suite is designed to run in CI/CD pipelines:

```yaml
# Example GitHub Actions configuration
- name: Run Tests
  run: |
    ./tests/run_tests.sh all -v

- name: Generate Coverage
  run: |
    ./tests/run_tests.sh coverage

- name: Upload Coverage
  uses: codecov/codecov-action@v2
  with:
    file: ./coverage/coverage.out
```

## Performance Benchmarks

Current performance benchmarks (reference values):

- **Single Message Send**: ~500µs/op
- **Multi-Target Send (10 targets)**: ~2ms/op
- **Concurrent Sends**: > 10,000 msg/sec
- **Message Creation**: ~1µs/op
- **Queue Throughput**: > 50,000 msg/sec
- **Memory Usage**: < 100MB for 10,000 messages

## Troubleshooting

### Common Issues

1. **Test Timeout**: Increase timeout values in test.env
2. **Race Conditions**: Run with `-race` flag: `go test -race ./tests/...`
3. **Memory Issues**: Check for leaks with pprof
4. **Flaky Tests**: Use `helper.AssertEventually()` for async operations

### Debug Mode

Enable debug logging for detailed output:

```bash
export DEBUG_MODE=true
export TEST_LOG_LEVEL=debug
./run_tests.sh unit -v
```

## Contributing

When adding new tests:

1. Follow existing patterns and naming conventions
2. Use the test helper utilities
3. Add appropriate documentation
4. Ensure tests are deterministic
5. Include both positive and negative test cases
6. Add benchmarks for performance-critical code

## Test Coverage Goals

- Unit Tests: > 80% coverage
- Integration Tests: Cover all critical paths
- Performance Tests: Benchmark all hot paths
- Overall: > 70% total coverage