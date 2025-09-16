# NotifyHub Kafka Pipeline - Testing Guide

Complete testing strategy and guide for the NotifyHub Kafka Pipeline examples.

## 📋 Overview

This guide covers comprehensive testing for the NotifyHub Kafka Pipeline, including:

- **Unit Tests** - Individual component testing with mocks
- **Integration Tests** - End-to-end pipeline testing
- **Load Tests** - Performance and scalability testing
- **Quality Checks** - Code formatting, vetting, and coverage

## 🧪 Test Structure

```
examples/
├── gin-kafka-producer/
│   ├── main_test.go          # Unit tests for HTTP service
│   └── coverage.html         # Coverage report (generated)
├── kafka-consumer-notifier/
│   ├── main_test.go          # Unit tests for consumer
│   └── coverage.html         # Coverage report (generated)
├── integration_test.go       # End-to-end pipeline tests
├── test_runner.sh           # Automated test execution
└── TESTING_GUIDE.md         # This file
```

## 🚀 Quick Start

### Run All Tests

```bash
cd examples
./test_runner.sh
```

### Run Specific Test Types

```bash
# Unit tests only
./test_runner.sh --unit-only

# Integration tests only
./test_runner.sh --integration-only

# Include load testing
./test_runner.sh --load-tests

# Generate coverage reports
./test_runner.sh --coverage

# Clean up after testing
./test_runner.sh --cleanup
```

## 📊 Unit Tests

### gin-kafka-producer Tests

Located in `gin-kafka-producer/main_test.go`:

#### Test Categories

1. **Message Building Tests**
   - `TestBuildNotificationMessage` - Validates NotifyHub message construction
   - Tests title, body, targets, variables, and metadata handling
   - Validates input validation and error cases

2. **Kafka Message Creation Tests**
   - `TestCreateKafkaMessage` - Tests Kafka message wrapper creation
   - Validates JSON serialization and message structure
   - Tests processing hints and metadata inclusion

3. **HTTP Handler Tests**
   - `TestSendNotificationHandler` - End-to-end HTTP request handling
   - Tests success cases, validation failures, and Kafka failures
   - Uses mock Kafka writer for isolated testing

4. **Utility Tests**
   - `TestHealthHandler` - Health check endpoint
   - `TestMetricsHandler` - Metrics collection and reporting
   - `TestStatusHandler` - Service status information
   - `TestValidateRequest` - Request validation logic
   - `TestCORSMiddleware` - CORS configuration

5. **Benchmark Tests**
   - `BenchmarkBuildNotificationMessage` - Message building performance
   - `BenchmarkCreateKafkaMessage` - Kafka message creation performance

#### Running gin-kafka-producer Tests

```bash
cd gin-kafka-producer

# Run all tests
go test -v ./...

# Run specific test
go test -run TestBuildNotificationMessage -v

# Run benchmarks
go test -bench=. -benchmem

# Generate coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### kafka-consumer-notifier Tests

Located in `kafka-consumer-notifier/main_test.go`:

#### Test Categories

1. **Message Processing Tests**
   - `TestProcessMessage` - Core message processing logic
   - Tests successful processing, partial failures, and error handling
   - Uses mock NotifyHub for isolated testing

2. **Kafka Message Handling**
   - `TestUnmarshalKafkaMessage` - JSON deserialization testing
   - Tests valid messages, invalid JSON, and edge cases

3. **Retry Logic Tests**
   - `TestRetryWithBackoff` - Exponential backoff retry mechanism
   - Tests success scenarios, failure scenarios, and context cancellation
   - `TestRetryWithBackoffContextCancellation` - Context timeout handling

4. **Consumer Lifecycle Tests**
   - `TestConsumerRun` - Main consumer loop behavior
   - `TestNewConsumer` - Consumer creation and configuration
   - `TestNewConsumerValidation` - Configuration validation

5. **Metrics and Utilities**
   - `TestMetricsUpdate` - Metrics collection and updates
   - `TestGetEnvWithDefault` - Environment variable handling

6. **Benchmark Tests**
   - `BenchmarkProcessMessage` - Message processing performance
   - `BenchmarkUnmarshalKafkaMessage` - JSON deserialization performance
   - `BenchmarkRetryWithBackoff` - Retry logic performance

#### Running kafka-consumer-notifier Tests

```bash
cd kafka-consumer-notifier

# Run all tests
go test -v ./...

# Run specific test
go test -run TestProcessMessage -v

# Run benchmarks
go test -bench=. -benchmem

# Generate coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## 🔗 Integration Tests

Located in `integration_test.go`:

### Test Suite: IntegrationTestSuite

#### Setup
- Real Kafka broker connection (requires Docker)
- NotifyHub with test configuration
- Mock HTTP server simulating gin-kafka-producer

#### Test Categories

1. **Complete Pipeline Tests**
   - `TestCompleteNotificationPipeline` - Full end-to-end workflow
   - Tests simple email notifications and multi-platform alerts
   - Validates message flow: HTTP → Kafka → NotifyHub → Results

2. **Error Handling Tests**
   - `TestErrorHandling` - Invalid request handling
   - Tests malformed JSON, missing fields, and validation failures

3. **Connectivity Tests**
   - `TestKafkaConnectivity` - Basic Kafka read/write operations
   - `TestNotifyHubConfiguration` - NotifyHub setup validation

4. **Performance Tests**
   - `TestConcurrentProcessing` - Concurrent message handling
   - `TestPerformance` - Basic performance measurements

#### Running Integration Tests

```bash
cd examples

# Start Kafka (required for integration tests)
docker-compose up -d zookeeper kafka

# Run integration tests
go test -v -tags=integration ./...

# Or use the test runner
./test_runner.sh --integration-only
```

## 📈 Load Tests

### Automated Load Testing

The `gin-kafka-producer/examples/load-test.sh` script provides comprehensive load testing:

#### Test Scenarios

1. **Sequential Load Test**
   - Sends messages one after another
   - Measures total throughput and success rate

2. **Concurrent Load Test**
   - Multiple worker processes sending simultaneously
   - Tests system behavior under concurrent load

3. **Sustained Load Test**
   - Continuous message sending for specified duration
   - Tests system stability over time

#### Load Test Configuration

```bash
# Default configuration in load-test.sh
BASE_URL="http://localhost:8080"
CONCURRENT_REQUESTS=10
TOTAL_REQUESTS=100
TEST_DURATION=60  # seconds
```

#### Running Load Tests

```bash
# Start the complete pipeline
docker-compose up -d

# Wait for services to be ready
sleep 30

# Run load tests
cd gin-kafka-producer/examples
./load-test.sh

# Or use the test runner
cd ../../
./test_runner.sh --load-tests
```

## 🎯 Quality Checks

### Code Formatting

```bash
# Check formatting
gofmt -l gin-kafka-producer/
gofmt -l kafka-consumer-notifier/

# Auto-format
gofmt -s -w gin-kafka-producer/
gofmt -s -w kafka-consumer-notifier/
```

### Code Vetting

```bash
# Run go vet
go vet ./gin-kafka-producer/...
go vet ./kafka-consumer-notifier/...
```

### Test Coverage

```bash
# Generate coverage for both services
cd gin-kafka-producer
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out  # Summary
go tool cover -html=coverage.out -o coverage.html  # HTML report

cd ../kafka-consumer-notifier
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out  # Summary
go tool cover -html=coverage.out -o coverage.html  # HTML report
```

## 🛠️ Test Infrastructure

### Mock Objects

#### gin-kafka-producer Mocks
- `MockKafkaWriter` - Mocks Kafka message publishing
- Used for isolated testing without real Kafka dependency

#### kafka-consumer-notifier Mocks
- `MockKafkaReader` - Mocks Kafka message consumption
- `MockNotifyHub` - Mocks NotifyHub client interface
- Enables testing without external dependencies

### Test Data Creation

#### Helper Functions
- `createTestKafkaMessage()` - Creates valid Kafka messages for testing
- Generates realistic test data with proper structure
- Used across multiple test cases for consistency

### Test Environment

#### Docker Configuration
- Uses `docker-compose.yml` for test environment
- Kafka and Zookeeper containers for integration testing
- Isolated test topics to prevent interference

## 📋 Test Execution Strategies

### Development Testing

```bash
# Quick unit tests during development
cd gin-kafka-producer && go test ./...
cd kafka-consumer-notifier && go test ./...

# Watch mode (with external tool like 'entr')
find . -name "*.go" | entr -r go test ./...
```

### CI/CD Pipeline Testing

```bash
# Complete test suite for CI/CD
./test_runner.sh --coverage --cleanup

# Fast feedback for pull requests
./test_runner.sh --unit-only --no-quality
```

### Pre-deployment Testing

```bash
# Full validation including load tests
./test_runner.sh --load-tests --coverage

# Performance validation
cd gin-kafka-producer/examples && ./load-test.sh
```

## 🔍 Debugging Tests

### Verbose Output

```bash
# Enable verbose test output
go test -v ./...

# Show test coverage during run
go test -v -cover ./...
```

### Test-specific Debugging

```bash
# Run single test with detailed output
go test -run TestSpecificTest -v

# Run with race detection
go test -race ./...

# Run with memory sanitizer
go test -msan ./...
```

### Kafka Debugging

```bash
# Check Kafka logs
docker-compose logs kafka

# Monitor Kafka topics
docker exec notifyhub-kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic notifications \
  --from-beginning

# Check consumer groups
docker exec notifyhub-kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --describe --group test-group
```

## 📊 Test Metrics and Reporting

### Coverage Targets

- **Unit Test Coverage**: Target 80%+ line coverage
- **Integration Coverage**: All major user journeys
- **Error Path Coverage**: All error conditions tested

### Performance Benchmarks

- **Message Building**: < 1ms per message
- **Kafka Operations**: < 10ms per message
- **End-to-end Latency**: < 100ms for simple notifications

### Quality Gates

- All tests must pass before merge
- Code coverage must not decrease
- Performance benchmarks must not regress
- No formatting or vetting issues

## 🚨 Common Issues and Solutions

### Kafka Connection Issues

```bash
# Check if Kafka is running
docker-compose ps kafka

# Restart Kafka if needed
docker-compose restart kafka

# Check network connectivity
docker-compose logs kafka
```

### Test Isolation Issues

```bash
# Clean test environment
docker-compose down
docker-compose up -d zookeeper kafka

# Use unique test topics
export TEST_TOPIC="test-$(date +%s)"
```

### Timing Issues in Tests

```bash
# Increase timeouts for slow environments
export TEST_TIMEOUT="30s"

# Use retries for flaky operations
# (already implemented in retry logic tests)
```

## 🔧 Customizing Tests

### Environment Variables

```bash
# Kafka configuration
export KAFKA_BROKERS="localhost:9092"
export KAFKA_TOPIC="test-notifications"

# Test configuration
export TEST_TIMEOUT="10s"
export TEST_WORKERS="4"

# Load test configuration
export LOAD_TEST_REQUESTS="50"
export LOAD_TEST_CONCURRENT="5"
```

### Test Data Customization

Modify test data in test files:
- Update message templates in `TestCompleteNotificationPipeline`
- Adjust load test parameters in `load-test.sh`
- Configure mock responses in mock setup functions

## 📚 Best Practices

### Test Organization
- Keep unit tests fast and isolated
- Use mocks for external dependencies
- Test both success and failure paths
- Include edge cases and boundary conditions

### Test Data
- Use realistic but anonymized test data
- Create reusable test fixtures
- Avoid hardcoded values where possible
- Clean up test data after tests

### Performance Testing
- Run performance tests in isolated environment
- Use consistent hardware for benchmarks
- Monitor resource usage during tests
- Set realistic performance expectations

### Maintenance
- Update tests when code changes
- Review test coverage regularly
- Refactor test code for clarity
- Document test intentions clearly

## 🎉 Success Criteria

A successful test run should show:

```
✅ Code quality checks: PASSED
✅ Unit tests: PASSED
✅ Integration tests: PASSED
✅ Load tests: COMPLETED
📊 Coverage reports: GENERATED

🚀 The NotifyHub Kafka Pipeline is ready for production!
```

This comprehensive testing ensures the NotifyHub Kafka Pipeline is reliable, performant, and ready for production deployment.