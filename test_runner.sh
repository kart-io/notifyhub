#!/bin/bash

# NotifyHub Test Runner
# This script runs comprehensive tests for the NotifyHub notification system

set -e

echo "🧪 NotifyHub Test Runner"
echo "======================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
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

print_section() {
    echo -e "\n${PURPLE}=== $1 ===${NC}"
}

# Function to check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    go_version=$(go version | awk '{print $3}')
    print_status "Using Go version: $go_version"
}

# Function to check Go module
check_module() {
    if [ ! -f "go.mod" ]; then
        print_error "go.mod not found. Please run this script from the NotifyHub root directory"
        exit 1
    fi
    
    module_name=$(grep "^module " go.mod | awk '{print $2}')
    print_status "Module: $module_name"
}

# Function to run unit tests
run_unit_tests() {
    print_section "Running Unit Tests"
    
    print_status "Running unit tests with coverage..."
    
    # Create coverage directory
    mkdir -p coverage
    
    # Run tests with coverage
    if go test -v -race -coverprofile=coverage/unit.out -covermode=atomic ./...; then
        print_success "Unit tests passed"
        
        # Generate coverage report
        go tool cover -html=coverage/unit.out -o coverage/unit.html
        
        # Get coverage percentage
        coverage_percent=$(go tool cover -func=coverage/unit.out | grep total | awk '{print $3}')
        print_success "Unit test coverage: $coverage_percent"
        
        return 0
    else
        print_error "Unit tests failed"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_section "Running Integration Tests"
    
    print_status "Running integration tests..."
    
    # Run integration tests (may take longer)
    if go test -v -race -tags=integration -timeout=10m ./...; then
        print_success "Integration tests passed"
        return 0
    else
        print_error "Integration tests failed"
        return 1
    fi
}

# Function to run E2E tests
run_e2e_tests() {
    print_section "Running E2E Tests"
    
    print_status "Running end-to-end tests..."
    
    # Run E2E tests with longer timeout
    if go test -v -race -run="TestE2ESuite" -timeout=15m .; then
        print_success "E2E tests passed"
        return 0
    else
        print_error "E2E tests failed"
        return 1
    fi
}

# Function to run performance tests
run_performance_tests() {
    print_section "Running Performance Tests"
    
    print_status "Running performance benchmarks..."
    
    # Run benchmarks
    if go test -v -bench=. -benchmem -timeout=30m -run=^$ .; then
        print_success "Performance tests completed"
        return 0
    else
        print_error "Performance tests failed"
        return 1
    fi
}

# Function to run load tests
run_load_tests() {
    print_section "Running Load Tests"
    
    print_status "Running load tests (this may take several minutes)..."
    
    # Run load tests
    if go test -v -run="TestLoad" -timeout=30m .; then
        print_success "Load tests completed"
        return 0
    else
        print_error "Load tests failed"
        return 1
    fi
}

# Function to run specific package tests
run_package_tests() {
    local package=$1
    print_section "Running Tests for Package: $package"
    
    if go test -v -race -coverprofile=coverage/${package//\//_}.out ./$package; then
        print_success "Tests passed for package: $package"
        
        # Generate coverage for this package
        go tool cover -html=coverage/${package//\//_}.out -o coverage/${package//\//_}.html
        
        return 0
    else
        print_error "Tests failed for package: $package"
        return 1
    fi
}

# Function to run code quality checks
run_quality_checks() {
    print_section "Running Code Quality Checks"
    
    # Format check
    print_status "Checking code formatting..."
    if ! gofmt -l . | grep -q .; then
        print_success "Code is properly formatted"
    else
        print_error "Code formatting issues found:"
        gofmt -l .
        return 1
    fi
    
    # Vet check
    print_status "Running go vet..."
    if go vet ./...; then
        print_success "go vet passed"
    else
        print_error "go vet failed"
        return 1
    fi
    
    # Mod tidy check
    print_status "Checking go mod tidy..."
    if go mod tidy && git diff --exit-code go.mod go.sum; then
        print_success "go.mod and go.sum are tidy"
    else
        print_error "go.mod or go.sum needs to be tidied"
        return 1
    fi
    
    return 0
}

# Function to generate test reports
generate_reports() {
    print_section "Generating Test Reports"
    
    if [ -d "coverage" ]; then
        print_status "Generating combined coverage report..."
        
        # Combine all coverage files
        echo "mode: atomic" > coverage/combined.out
        for file in coverage/*.out; do
            if [ "$file" != "coverage/combined.out" ] && [ -f "$file" ]; then
                tail -n +2 "$file" >> coverage/combined.out
            fi
        done
        
        # Generate combined HTML report
        go tool cover -html=coverage/combined.out -o coverage/combined.html
        
        # Get combined coverage
        combined_coverage=$(go tool cover -func=coverage/combined.out | grep total | awk '{print $3}')
        print_success "Combined test coverage: $combined_coverage"
        
        print_status "Coverage reports generated in coverage/ directory:"
        ls -la coverage/*.html
    fi
}

# Function to run validation tests
run_validation_tests() {
    print_section "Running Validation Tests"
    
    print_status "Running architecture validation tests..."
    
    if go test -v -run="TestValidation" ./_tests/; then
        print_success "Validation tests passed"
        return 0
    else
        print_error "Validation tests failed"
        return 1
    fi
}

# Function to run examples tests
run_examples_tests() {
    print_section "Running Examples Tests"
    
    if [ -d "examples" ]; then
        print_status "Running example tests..."
        
        cd examples
        
        # Run example-specific tests if they exist
        if [ -f "test_runner.sh" ]; then
            print_status "Running examples test runner..."
            if ./test_runner.sh --unit-only; then
                print_success "Examples tests passed"
                cd ..
                return 0
            else
                print_error "Examples tests failed"
                cd ..
                return 1
            fi
        else
            print_warning "No examples test runner found"
            cd ..
            return 0
        fi
    else
        print_warning "No examples directory found"
        return 0
    fi
}

# Function to clean up test artifacts
cleanup() {
    print_section "Cleaning Up"
    
    print_status "Cleaning up test artifacts..."
    
    # Remove temporary test files
    find . -name "*.test" -delete
    find . -name "test.out" -delete
    
    print_success "Cleanup completed"
}

# Function to show usage
show_usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --unit              Run unit tests only"
    echo "  --integration       Run integration tests only"
    echo "  --e2e               Run E2E tests only"
    echo "  --performance       Run performance tests only"
    echo "  --load              Run load tests only"
    echo "  --quality           Run code quality checks only"
    echo "  --validation        Run validation tests only"
    echo "  --examples          Run examples tests only"
    echo "  --package <name>    Run tests for specific package"
    echo "  --fast              Run fast tests (unit + quality)"
    echo "  --full              Run full test suite (default)"
    echo "  --no-coverage       Skip coverage generation"
    echo "  --no-reports        Skip report generation"
    echo "  --cleanup           Clean up test artifacts only"
    echo "  --help              Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                          # Run full test suite"
    echo "  $0 --fast                   # Run fast tests only"
    echo "  $0 --unit --no-coverage     # Run unit tests without coverage"
    echo "  $0 --package client         # Test client package only"
    echo "  $0 --quality                # Run code quality checks only"
}

# Main execution
main() {
    local run_unit=false
    local run_integration=false
    local run_e2e=false
    local run_performance=false
    local run_load=false
    local run_quality=false
    local run_validation=false
    local run_examples=false
    local run_package=""
    local generate_coverage=true
    local generate_reports=true
    local run_cleanup=false
    local mode="full"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --unit)
                run_unit=true
                mode="custom"
                shift
                ;;
            --integration)
                run_integration=true
                mode="custom"
                shift
                ;;
            --e2e)
                run_e2e=true
                mode="custom"
                shift
                ;;
            --performance)
                run_performance=true
                mode="custom"
                shift
                ;;
            --load)
                run_load=true
                mode="custom"
                shift
                ;;
            --quality)
                run_quality=true
                mode="custom"
                shift
                ;;
            --validation)
                run_validation=true
                mode="custom"
                shift
                ;;
            --examples)
                run_examples=true
                mode="custom"
                shift
                ;;
            --package)
                run_package="$2"
                mode="custom"
                shift 2
                ;;
            --fast)
                run_unit=true
                run_quality=true
                mode="custom"
                shift
                ;;
            --full)
                mode="full"
                shift
                ;;
            --no-coverage)
                generate_coverage=false
                shift
                ;;
            --no-reports)
                generate_reports=false
                shift
                ;;
            --cleanup)
                run_cleanup=true
                mode="custom"
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # Run cleanup only if requested
    if [ "$run_cleanup" = true ]; then
        cleanup
        exit 0
    fi
    
    # Check prerequisites
    check_go
    check_module
    
    # Set full mode defaults
    if [ "$mode" = "full" ]; then
        run_unit=true
        run_integration=true
        run_e2e=true
        run_quality=true
        run_validation=true
        run_examples=true
    fi
    
    # Track test results
    local test_results=()
    local overall_success=true
    
    # Create coverage directory if needed
    if [ "$generate_coverage" = true ]; then
        mkdir -p coverage
    fi
    
    # Run tests based on options
    if [ "$run_quality" = true ]; then
        if run_quality_checks; then
            test_results+=("✅ Code Quality: PASSED")
        else
            test_results+=("❌ Code Quality: FAILED")
            overall_success=false
        fi
    fi
    
    if [ "$run_unit" = true ]; then
        if run_unit_tests; then
            test_results+=("✅ Unit Tests: PASSED")
        else
            test_results+=("❌ Unit Tests: FAILED")
            overall_success=false
        fi
    fi
    
    if [ "$run_validation" = true ]; then
        if run_validation_tests; then
            test_results+=("✅ Validation Tests: PASSED")
        else
            test_results+=("❌ Validation Tests: FAILED")
            overall_success=false
        fi
    fi
    
    if [ "$run_integration" = true ]; then
        if run_integration_tests; then
            test_results+=("✅ Integration Tests: PASSED")
        else
            test_results+=("❌ Integration Tests: FAILED")
            overall_success=false
        fi
    fi
    
    if [ "$run_e2e" = true ]; then
        if run_e2e_tests; then
            test_results+=("✅ E2E Tests: PASSED")
        else
            test_results+=("❌ E2E Tests: FAILED")
            overall_success=false
        fi
    fi
    
    if [ "$run_examples" = true ]; then
        if run_examples_tests; then
            test_results+=("✅ Examples Tests: PASSED")
        else
            test_results+=("❌ Examples Tests: FAILED")
            overall_success=false
        fi
    fi
    
    if [ -n "$run_package" ]; then
        if run_package_tests "$run_package"; then
            test_results+=("✅ Package Tests ($run_package): PASSED")
        else
            test_results+=("❌ Package Tests ($run_package): FAILED")
            overall_success=false
        fi
    fi
    
    if [ "$run_performance" = true ]; then
        if run_performance_tests; then
            test_results+=("✅ Performance Tests: COMPLETED")
        else
            test_results+=("❌ Performance Tests: FAILED")
            overall_success=false
        fi
    fi
    
    if [ "$run_load" = true ]; then
        if run_load_tests; then
            test_results+=("✅ Load Tests: COMPLETED")
        else
            test_results+=("❌ Load Tests: FAILED")
            overall_success=false
        fi
    fi
    
    # Generate reports
    if [ "$generate_reports" = true ] && [ "$generate_coverage" = true ]; then
        generate_reports
    fi
    
    # Print summary
    print_section "Test Summary"
    for result in "${test_results[@]}"; do
        echo "  $result"
    done
    
    echo ""
    if [ "$overall_success" = true ]; then
        print_success "🎉 All tests passed!"
        echo ""
        echo "📊 Reports generated:"
        if [ -d "coverage" ]; then
            echo "  - Coverage reports: coverage/*.html"
        fi
        echo ""
        echo "🚀 NotifyHub is ready for deployment!"
        exit 0
    else
        print_error "❌ Some tests failed!"
        echo ""
        echo "Please review the test output above and fix any issues."
        exit 1
    fi
}

# Run main function with all arguments
main "$@"