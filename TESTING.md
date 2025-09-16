# NotifyHub Testing Guide

Complete testing strategy and documentation for the NotifyHub notification system.

## 📋 Overview

This guide covers comprehensive testing for NotifyHub, including:

- **Unit Tests** - Component-level testing with mocks and stubs
- **Integration Tests** - Multi-component workflow testing
- **E2E Tests** - Complete end-to-end system testing
- **Performance Tests** - Benchmarking and load testing
- **Quality Checks** - Code formatting, vetting, and static analysis

## 🧪 Test Structure

```
notifyhub/
├── *_test.go                    # Unit tests alongside source code
├── e2e_test.go                  # End-to-end integration tests
├── performance_test.go          # Performance and load tests
├── test_runner.sh              # Automated test execution script
├── coverage/                   # Generated coverage reports
├── config/
│   ├── config_test.go          # Configuration testing
│   └── routing_test.go         # Routing engine testing
├── client/
│   └── hub_test.go             # Hub and client testing
├── notifiers/
│   └── notifiers_test.go       # Notifier implementations testing
├── queue/
│   └── queue_test.go           # Queue systems testing
├── internal/
│   └── internal_test.go        # Internal utilities testing
├── template/
│   └── template_test.go        # Template engine testing
├── monitoring/
│   └── monitoring_test.go      # Metrics and monitoring testing
├── examples/
│   ├── gin-kafka-producer/
│   │   └── main_test.go        # Producer service testing
│   ├── kafka-consumer-notifier/
│   │   └── main_test.go        # Consumer service testing
│   └── integration_test.go     # Pipeline integration testing
└── _tests/
    ├── integration_test.go     # Legacy integration tests
    └── validation_test.go      # Architecture validation tests
```

## 🚀 Quick Start

### Run All Tests

```bash
./test_runner.sh
```

### Run Specific Test Types

```bash
# Fast tests (unit + quality checks)
./test_runner.sh --fast

# Unit tests only
./test_runner.sh --unit

# Integration and E2E tests
./test_runner.sh --integration --e2e

# Performance benchmarks
./test_runner.sh --performance

# Load testing
./test_runner.sh --load

# Code quality checks
./test_runner.sh --quality
```

### Run Package-Specific Tests

```bash
# Test specific package
./test_runner.sh --package client
./test_runner.sh --package config
./test_runner.sh --package notifiers

# Test with coverage
go test -v -race -coverprofile=coverage.out ./client
go tool cover -html=coverage.out -o coverage.html
```

## 📊 Unit Tests

### Overview

Unit tests validate individual components in isolation using mocks and stubs for external dependencies.

#### Test Categories

1. **Configuration Tests** (`config/config_test.go`)
   - Environment variable parsing
   - Configuration validation
   - Option builders and combiners
   - Default value handling

2. **Routing Tests** (`config/routing_test.go`)
   - Rule creation and validation
   - Message matching logic
   - Routing engine behavior
   - Priority-based routing

3. **Hub Tests** (`client/hub_test.go`)
   - Hub lifecycle (creation, start, stop)
   - Message sending (sync, async, batch)
   - Health checks and metrics
   - Error handling and validation

4. **Notifier Tests** (`notifiers/notifiers_test.go`)
   - Notifier interface compliance
   - Target support validation
   - Message format handling
   - Platform-specific logic

5. **Queue Tests** (`queue/queue_test.go`)
   - Queue operations (enqueue, dequeue)
   - Worker pool management
   - Retry policies and backoff
   - Callback system

6. **Internal Utilities Tests** (`internal/internal_test.go`)
   - ID generation and uniqueness
   - Rate limiting algorithms
   - Utility functions

7. **Template Tests** (`template/template_test.go`)
   - Template parsing and rendering
   - Variable substitution
   - Custom functions
   - Error handling

8. **Monitoring Tests** (`monitoring/monitoring_test.go`)
   - Metrics collection (counters, gauges, histograms)
   - Thread safety
   - Performance tracking

#### Running Unit Tests

```bash
# Run all unit tests with coverage
go test -v -race -coverprofile=coverage/unit.out -covermode=atomic ./...

# Run specific package tests
go test -v -race ./config
go test -v -race ./client
go test -v -race ./notifiers

# Run with short mode (skip slow tests)
go test -short ./...

# Run specific test
go test -run TestHubCreation ./client

# Generate coverage report
go tool cover -html=coverage/unit.out -o coverage/unit.html
```

#### Test Patterns

**Mocking External Dependencies:**
```go
// Mock notifier for testing
type MockNotifier struct {
    mock.Mock
}

func (m *MockNotifier) Send(ctx context.Context, message *notifiers.Message) ([]notifiers.SendResult, error) {
    args := m.Called(ctx, message)
    return args.Get(0).([]notifiers.SendResult), args.Error(1)
}
```

**Table-Driven Tests:**
```go
tests := []struct {
    name    string
    input   interface{}
    want    interface{}
    wantErr bool
}{
    {
        name:    "valid input",
        input:   validInput,
        want:    expectedOutput,
        wantErr: false,
    },
    // ... more test cases
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test implementation
    })
}
```

## 🔗 Integration Tests

### Overview

Integration tests validate interactions between multiple components without external dependencies.

#### Test Suites

1. **Legacy Integration Tests** (`_tests/integration_test.go`)
   - Basic workflow validation
   - Configuration integration
   - Simple end-to-end scenarios

2. **Validation Tests** (`_tests/validation_test.go`)
   - Architecture compliance
   - Interface implementations
   - Component relationships

#### Running Integration Tests

```bash
# Run integration tests
go test -v -race -tags=integration ./...

# Run validation tests
go test -v -run="TestValidation" ./_tests/

# Run with timeout for long-running tests
go test -v -timeout=10m -tags=integration ./...
```

## 🎯 E2E Tests

### Overview

End-to-end tests validate complete system behavior with real component interactions but mock external services.

#### Test Suite: E2ETestSuite (`e2e_test.go`)

**Test Categories:**

1. **Hub Lifecycle**
   - Hub creation and configuration
   - Start/stop lifecycle
   - Health and metrics

2. **Message Builders**
   - All message builder types
   - Target configuration
   - Variable and metadata handling

3. **Sending Methods**
   - Synchronous sending
   - Asynchronous sending
   - Batch sending
   - Convenience methods

4. **Error Handling**
   - Invalid input handling
   - Context cancellation
   - Network failures

5. **Concurrent Operations**
   - Multiple goroutines
   - Race condition detection
   - Performance under load

6. **Queue Integration**
   - Async processing
   - Delayed messages
   - Worker pool behavior

#### Running E2E Tests

```bash
# Run E2E test suite
go test -v -race -run="TestE2ESuite" -timeout=15m .

# Run specific E2E test
go test -v -run="TestE2ESuite/TestHubLifecycle" .

# Run with custom timeout
go test -v -timeout=30m -run="TestE2ESuite" .
```

## 📈 Performance Tests

### Overview

Performance tests benchmark system components and identify bottlenecks.

#### Benchmark Categories (`performance_test.go`)

1. **Component Benchmarks**
   - `BenchmarkMessageBuilder` - Message creation performance
   - `BenchmarkHubCreation` - Hub initialization overhead
   - `BenchmarkSyncSend` - Synchronous sending throughput
   - `BenchmarkAsyncSend` - Asynchronous sending throughput

2. **Concurrent Benchmarks**
   - `BenchmarkConcurrentSend` - Multi-goroutine performance
   - Thread safety validation
   - Contention analysis

3. **Load Testing**
   - `TestLoadSyncSending` - High-volume synchronous sending
   - `TestLoadAsyncSending` - High-volume asynchronous sending
   - `TestSustainedLoad` - Long-duration load testing

#### Running Performance Tests

```bash
# Run all benchmarks
go test -v -bench=. -benchmem -timeout=30m -run=^$ .

# Run specific benchmark
go test -bench=BenchmarkMessageBuilder -benchmem .

# Run with custom iterations
go test -bench=BenchmarkSyncSend -benchtime=10s .

# Run load tests
go test -v -run="TestLoad" -timeout=30m .

# Profile CPU usage
go test -bench=BenchmarkConcurrentSend -cpuprofile=cpu.prof .
go tool pprof cpu.prof
```

#### Performance Targets

- **Message Builder**: < 1ms per message
- **Sync Send Throughput**: > 10 msg/sec
- **Async Send Throughput**: > 100 msg/sec
- **Memory Usage**: No significant leaks under load
- **Concurrent Safety**: No race conditions

## 🔧 Quality Checks

### Code Formatting

```bash
# Check formatting
gofmt -l .

# Auto-format code
gofmt -s -w .

# Check with goimports
goimports -l .
```

### Static Analysis

```bash
# Run go vet
go vet ./...

# Check module tidiness
go mod tidy
go mod verify

# Run with race detector
go test -race ./...
```

### Coverage Analysis

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Check coverage by function
go tool cover -func=coverage.out

# Set coverage threshold
go test -coverprofile=coverage.out ./...
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$COVERAGE < 80" | bc -l) )); then
    echo "Coverage $COVERAGE% is below 80% threshold"
    exit 1
fi
```

## 🛠️ Test Infrastructure

### Mock Objects

#### NotifyHub Mocks
- `MockNotifier` - Mock notification platform
- `MockQueue` - Mock queue implementation
- `MockLogger` - Mock logging interface

#### Kafka Pipeline Mocks
- `MockKafkaWriter` - Mock Kafka producer
- `MockKafkaReader` - Mock Kafka consumer
- `MockNotifyHub` - Mock NotifyHub client

### Test Data Factories

```go
// Create test message
func createTestMessage() *notifiers.Message {
    return client.NewMessage().
        Title("Test Message").
        Body("Test Body").
        AddTarget("email", "test@example.com").
        Build()
}

// Create test configuration
func createTestConfig() []func(*config.Options) error {
    return []func(*config.Options) error{
        config.WithTestDefaults(),
        config.WithFeishu("https://test.webhook", "secret"),
    }
}
```

### Test Utilities

```go
// Wait for async operation with timeout
func waitForOperation(condition func() bool, timeout time.Duration) bool {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if condition() {
            return true
        }
        time.Sleep(10 * time.Millisecond)
    }
    return false
}

// Assert eventually true
func assertEventually(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
    if !waitForOperation(condition, timeout) {
        t.Fatalf("Condition not met within %v: %s", timeout, msg)
    }
}
```

## 📋 Test Execution Strategies

### Development Testing

```bash
# Quick feedback during development
go test ./...

# Watch mode (with external tool)
find . -name "*.go" | entr -r go test ./...

# Test specific functionality
go test -run TestHubCreation ./client
```

### CI/CD Pipeline Testing

```bash
# Complete validation pipeline
./test_runner.sh --full

# Fast feedback for pull requests
./test_runner.sh --fast

# Performance regression testing
./test_runner.sh --performance
```

### Pre-deployment Testing

```bash
# Full validation including load tests
./test_runner.sh --full --load

# Performance baseline validation
./test_runner.sh --performance --load
```

## 🔍 Debugging Tests

### Verbose Output

```bash
# Enable verbose test output
go test -v ./...

# Show test coverage during run
go test -v -cover ./...

# Show detailed benchmark output
go test -v -bench=. -benchmem ./...
```

### Race Detection

```bash
# Run with race detector
go test -race ./...

# Combine with other flags
go test -v -race -cover ./...
```

### Memory Debugging

```bash
# Run with memory sanitizer (if available)
go test -msan ./...

# Profile memory usage
go test -bench=BenchmarkName -memprofile=mem.prof .
go tool pprof mem.prof
```

### Test-specific Debugging

```bash
# Run single test with detailed output
go test -run TestSpecificTest -v

# Set custom timeouts
go test -timeout=30m -run TestLongRunning

# Enable debug logging
DEBUG=true go test -v ./...
```

## 📊 Test Metrics and Reporting

### Coverage Targets

- **Overall Coverage**: Target 80%+ line coverage
- **Critical Components**: Target 90%+ coverage for core logic
- **Integration Coverage**: All major user journeys tested
- **Error Path Coverage**: All error conditions validated

### Performance Baselines

| Component | Metric | Target |
|-----------|---------|---------|
| Message Builder | Latency | < 1ms |
| Sync Send | Throughput | > 10 msg/sec |
| Async Send | Throughput | > 100 msg/sec |
| Queue Operations | Latency | < 10ms |
| Health Checks | Latency | < 100ms |

### Quality Gates

- ✅ All unit tests must pass
- ✅ Code coverage must not decrease
- ✅ No race conditions detected
- ✅ No formatting violations
- ✅ go vet must pass clean
- ✅ Performance benchmarks must not regress
- ✅ Load tests must complete successfully

## 🚨 Common Issues and Solutions

### Test Failures

#### Timeout Issues
```bash
# Increase test timeout
go test -timeout=30m ./...

# Check for deadlocks
go test -race -timeout=10m ./...
```

#### Race Conditions
```bash
# Always run with race detector
go test -race ./...

# Focus on concurrent tests
go test -race -run Concurrent ./...
```

#### Flaky Tests
```bash
# Run tests multiple times
go test -count=10 ./...

# Focus on specific flaky test
go test -count=100 -run TestFlaky ./...
```

### Performance Issues

#### Memory Leaks
```bash
# Profile memory usage
go test -bench=BenchmarkName -memprofile=mem.prof .
go tool pprof mem.prof

# Check for goroutine leaks
go test -run TestName -memprofile=mem.prof .
```

#### CPU Bottlenecks
```bash
# Profile CPU usage
go test -bench=BenchmarkName -cpuprofile=cpu.prof .
go tool pprof cpu.prof
```

### Coverage Issues

#### Low Coverage
```bash
# Identify uncovered code
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Focus on specific package
go test -coverprofile=coverage.out ./package
```

#### False Coverage
```bash
# Use atomic coverage mode
go test -covermode=atomic -coverprofile=coverage.out ./...

# Exclude generated files
go test -coverprofile=coverage.out ./... | grep -v generated
```

## 🎯 Best Practices

### Test Organization

- **Single Responsibility**: Each test validates one specific behavior
- **Clear Naming**: Test names describe what is being tested
- **Arrange-Act-Assert**: Structure tests with clear phases
- **Independent Tests**: Tests should not depend on each other

### Test Data

- **Realistic Data**: Use realistic but safe test data
- **Data Factories**: Create reusable test data builders
- **Avoid Hardcoding**: Use variables for test configuration
- **Clean State**: Ensure clean test state before each test

### Mock Usage

- **Interface-Based**: Mock interfaces, not concrete types
- **Behavior Verification**: Verify mock interactions
- **Minimal Mocking**: Only mock external dependencies
- **Clear Expectations**: Make mock expectations explicit

### Performance Testing

- **Baseline Measurements**: Establish performance baselines
- **Consistent Environment**: Run benchmarks in consistent conditions
- **Resource Monitoring**: Monitor CPU, memory, and I/O usage
- **Realistic Load**: Use realistic load patterns

## 📚 Additional Resources

### Documentation

- [Go Testing Package](https://pkg.go.dev/testing)
- [Testify Documentation](https://github.com/stretchr/testify)
- [Go Race Detector](https://golang.org/doc/articles/race_detector.html)
- [Go Coverage Tool](https://golang.org/cmd/cover/)

### Tools

- **testify/suite** - Test suite framework
- **testify/mock** - Mock generation and verification
- **testify/assert** - Rich assertion library
- **go tool cover** - Coverage analysis
- **go tool pprof** - Performance profiling

### External Testing

- **Postman/Newman** - API testing for examples
- **Artillery/k6** - Load testing for HTTP endpoints
- **Docker Compose** - Integration environment setup

## 🎉 Success Criteria

A successful test run should show:

```
🧪 NotifyHub Test Runner
=======================

=== Code Quality ===
✅ Code Quality: PASSED

=== Unit Tests ===
✅ Unit Tests: PASSED
✅ Unit test coverage: 85.3%

=== Validation Tests ===
✅ Validation Tests: PASSED

=== Integration Tests ===
✅ Integration Tests: PASSED

=== E2E Tests ===
✅ E2E Tests: PASSED

=== Examples Tests ===
✅ Examples Tests: PASSED

=== Performance Tests ===
✅ Performance Tests: COMPLETED

📊 Reports generated:
  - Coverage reports: coverage/*.html

🚀 NotifyHub is ready for deployment!
```

This comprehensive testing ensures NotifyHub is production-ready with high reliability, performance, and maintainability!