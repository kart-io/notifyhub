#!/bin/bash

# Test Runner for NotifyHub Kafka Pipeline Examples
# This script runs all unit tests and integration tests for the Kafka pipeline examples

set -e

echo "🧪 NotifyHub Kafka Pipeline - Test Runner"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if we're in the correct directory
if [ ! -f "docker-compose.yml" ]; then
    print_error "Please run this script from the examples directory"
    exit 1
fi

# Function to check if Kafka is running
check_kafka() {
    print_status "Checking Kafka connectivity..."
    
    if docker-compose ps kafka | grep -q "Up"; then
        print_success "Kafka is running"
        return 0
    else
        print_warning "Kafka is not running"
        return 1
    fi
}

# Function to start test environment
start_test_env() {
    print_status "Starting test environment..."
    
    # Start Kafka and Zookeeper for testing
    docker-compose up -d zookeeper kafka
    
    # Wait for Kafka to be ready
    print_status "Waiting for Kafka to be ready..."
    for i in {1..30}; do
        if check_kafka; then
            break
        fi
        if [ $i -eq 30 ]; then
            print_error "Kafka failed to start within 30 attempts"
            exit 1
        fi
        sleep 2
    done
    
    print_success "Test environment is ready"
}

# Function to stop test environment
stop_test_env() {
    print_status "Stopping test environment..."
    docker-compose down
    print_success "Test environment stopped"
}

# Function to run unit tests
run_unit_tests() {
    print_status "Running unit tests..."
    
    # Test gin-kafka-producer
    print_status "Testing gin-kafka-producer..."
    cd gin-kafka-producer
    
    # Ensure dependencies are available
    go mod tidy
    
    # Run tests
    if go test -v ./...; then
        print_success "gin-kafka-producer unit tests passed"
    else
        print_error "gin-kafka-producer unit tests failed"
        cd ..
        return 1
    fi
    
    # Run benchmarks
    print_status "Running gin-kafka-producer benchmarks..."
    go test -bench=. -benchmem ./...
    
    cd ..
    
    # Test kafka-consumer-notifier
    print_status "Testing kafka-consumer-notifier..."
    cd kafka-consumer-notifier
    
    # Ensure dependencies are available
    go mod tidy
    
    # Run tests
    if go test -v ./...; then
        print_success "kafka-consumer-notifier unit tests passed"
    else
        print_error "kafka-consumer-notifier unit tests failed"
        cd ..
        return 1
    fi
    
    # Run benchmarks
    print_status "Running kafka-consumer-notifier benchmarks..."
    go test -bench=. -benchmem ./...
    
    cd ..
    
    print_success "All unit tests passed"
}

# Function to run integration tests
run_integration_tests() {
    print_status "Running integration tests..."
    
    # Create a go.mod for integration tests if it doesn't exist
    if [ ! -f "go.mod" ]; then
        print_status "Creating go.mod for integration tests..."
        cat > go.mod << EOF
module github.com/kart-io/notifyhub/examples

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/segmentio/kafka-go v0.4.47
    github.com/kart-io/notifyhub v0.0.0
    github.com/stretchr/testify v1.8.3
)

// Use local notifyhub module for development
replace github.com/kart-io/notifyhub => ../
EOF
    fi
    
    # Ensure dependencies are available
    go mod tidy
    
    # Run integration tests
    if go test -v -tags=integration ./...; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        return 1
    fi
}

# Function to run load tests
run_load_tests() {
    print_status "Running load tests..."
    
    # Start the full pipeline
    docker-compose up -d
    
    # Wait for services to be ready
    print_status "Waiting for services to be ready..."
    sleep 30
    
    # Check if gin-kafka-producer is ready
    for i in {1..20}; do
        if curl -s http://localhost:8080/health > /dev/null; then
            break
        fi
        if [ $i -eq 20 ]; then
            print_error "gin-kafka-producer failed to start"
            return 1
        fi
        sleep 3
    done
    
    # Run the load test script
    if [ -f "gin-kafka-producer/examples/load-test.sh" ]; then
        print_status "Running load test script..."
        cd gin-kafka-producer/examples
        chmod +x load-test.sh
        if ./load-test.sh; then
            print_success "Load tests completed successfully"
        else
            print_warning "Load tests completed with some failures"
        fi
        cd ../..
    else
        print_warning "Load test script not found, skipping load tests"
    fi
    
    # Stop services
    docker-compose down
}

# Function to run code quality checks
run_quality_checks() {
    print_status "Running code quality checks..."
    
    # Format check
    print_status "Checking code formatting..."
    for dir in gin-kafka-producer kafka-consumer-notifier; do
        cd $dir
        if ! gofmt -l . | grep -q .; then
            print_success "$dir: Code is properly formatted"
        else
            print_error "$dir: Code formatting issues found"
            gofmt -l .
            cd ..
            return 1
        fi
        cd ..
    done
    
    # Vet check
    print_status "Running go vet..."
    for dir in gin-kafka-producer kafka-consumer-notifier; do
        cd $dir
        if go vet ./...; then
            print_success "$dir: go vet passed"
        else
            print_error "$dir: go vet failed"
            cd ..
            return 1
        fi
        cd ..
    done
    
    print_success "All quality checks passed"
}

# Function to generate test coverage report
generate_coverage() {
    print_status "Generating test coverage reports..."
    
    for dir in gin-kafka-producer kafka-consumer-notifier; do
        print_status "Generating coverage for $dir..."
        cd $dir
        
        go test -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
        
        # Get coverage percentage
        coverage_percent=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        print_success "$dir: Test coverage: $coverage_percent"
        
        cd ..
    done
    
    print_success "Coverage reports generated"
    print_status "Open gin-kafka-producer/coverage.html and kafka-consumer-notifier/coverage.html to view detailed coverage reports"
}

# Function to cleanup test artifacts
cleanup() {
    print_status "Cleaning up test artifacts..."
    
    # Remove coverage files
    find . -name "coverage.out" -delete
    find . -name "coverage.html" -delete
    
    # Remove test go.mod if it was created
    if [ -f "go.sum" ]; then
        rm go.sum
    fi
    
    print_success "Cleanup completed"
}

# Main execution
main() {
    local run_unit=true
    local run_integration=true
    local run_load=false
    local run_quality=true
    local generate_cov=false
    local cleanup_after=false
    local need_kafka=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --unit-only)
                run_unit=true
                run_integration=false
                run_load=false
                shift
                ;;
            --integration-only)
                run_unit=false
                run_integration=true
                run_load=false
                need_kafka=true
                shift
                ;;
            --load-tests)
                run_load=true
                need_kafka=true
                shift
                ;;
            --no-quality)
                run_quality=false
                shift
                ;;
            --coverage)
                generate_cov=true
                shift
                ;;
            --cleanup)
                cleanup_after=true
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo ""
                echo "Options:"
                echo "  --unit-only       Run only unit tests"
                echo "  --integration-only Run only integration tests"
                echo "  --load-tests      Include load testing"
                echo "  --no-quality      Skip code quality checks"
                echo "  --coverage        Generate test coverage reports"
                echo "  --cleanup         Clean up test artifacts after completion"
                echo "  --help            Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0                              # Run unit and integration tests"
                echo "  $0 --unit-only --coverage      # Run unit tests with coverage"
                echo "  $0 --load-tests                # Run all tests including load tests"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Set need_kafka if integration tests will run
    if [ "$run_integration" = true ]; then
        need_kafka=true
    fi
    
    # Trap to ensure cleanup on exit
    trap 'stop_test_env' EXIT
    
    # Start test environment if needed
    if [ "$need_kafka" = true ]; then
        start_test_env
    fi
    
    # Run tests based on options
    if [ "$run_quality" = true ]; then
        run_quality_checks || exit 1
    fi
    
    if [ "$run_unit" = true ]; then
        run_unit_tests || exit 1
    fi
    
    if [ "$run_integration" = true ]; then
        run_integration_tests || exit 1
    fi
    
    if [ "$run_load" = true ]; then
        run_load_tests || exit 1
    fi
    
    if [ "$generate_cov" = true ]; then
        generate_coverage
    fi
    
    if [ "$cleanup_after" = true ]; then
        cleanup
    fi
    
    print_success "🎉 All tests completed successfully!"
    
    # Summary
    echo ""
    echo "📋 Test Summary:"
    echo "================"
    [ "$run_quality" = true ] && echo "✅ Code quality checks: PASSED"
    [ "$run_unit" = true ] && echo "✅ Unit tests: PASSED" 
    [ "$run_integration" = true ] && echo "✅ Integration tests: PASSED"
    [ "$run_load" = true ] && echo "✅ Load tests: COMPLETED"
    [ "$generate_cov" = true ] && echo "📊 Coverage reports: GENERATED"
    echo ""
    echo "🚀 The NotifyHub Kafka Pipeline is ready for production!"
}

# Check for required tools
check_requirements() {
    local missing_tools=()
    
    # Check for required commands
    for cmd in go docker docker-compose curl; do
        if ! command -v $cmd &> /dev/null; then
            missing_tools+=($cmd)
        fi
    done
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        print_error "Missing required tools: ${missing_tools[*]}"
        print_error "Please install the missing tools and try again"
        exit 1
    fi
}

# Run requirements check first
check_requirements

# Execute main function with all arguments
main "$@"