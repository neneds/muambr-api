#!/bin/bash

# Test Runner Script for muambr-api
# This script provides convenient commands to run different types of tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to run tests with coverage
run_tests_with_coverage() {
    local test_path=$1
    local coverage_file=$2
    
    print_status "Running tests in $test_path with coverage..."
    go test -race -coverprofile="$coverage_file" -covermode=atomic "$test_path"
    
    if [ -f "$coverage_file" ]; then
        print_status "Generating coverage report..."
        go tool cover -html="$coverage_file" -o "${coverage_file%.out}.html"
        print_status "Coverage report generated: ${coverage_file%.out}.html"
    fi
}

# Parse command line arguments
case "${1:-all}" in
    "unit")
        print_status "Running unit tests..."
        go test -race -v ./tests/unit/...
        ;;
    "integration")
        print_status "Running integration tests..."
        export INTEGRATION_TESTS=true
        go test -race -v ./tests/integration/...
        ;;
    "extractors")
        print_status "Running extractor tests..."
        go test -race -v ./tests/unit/extractors/... ./tests/integration/extractors/...
        ;;
    "coverage")
        print_status "Running all tests with coverage..."
        mkdir -p coverage
        run_tests_with_coverage "./tests/unit/..." "coverage/unit.out"
        run_tests_with_coverage "./tests/integration/..." "coverage/integration.out"
        ;;
    "bench")
        print_status "Running benchmarks..."
        go test -bench=. -benchmem ./tests/...
        ;;
    "clean")
        print_status "Cleaning test artifacts..."
        rm -rf coverage/
        find . -name "*.test" -delete
        find . -name "*.out" -delete
        find . -name "*.html" -delete
        print_status "Test artifacts cleaned"
        ;;
    "all")
        print_status "Running all tests..."
        go test -race -v ./tests/unit/...
        
        print_warning "Integration tests require INTEGRATION_TESTS=true"
        print_status "To run integration tests: ./run_tests.sh integration"
        ;;
    "help"|"-h"|"--help")
        echo "Usage: $0 [COMMAND]"
        echo ""
        echo "Commands:"
        echo "  unit         Run unit tests only"
        echo "  integration  Run integration tests only (requires INTEGRATION_TESTS=true)"
        echo "  extractors   Run all extractor tests (unit + integration)"
        echo "  coverage     Run tests with coverage report generation"
        echo "  bench        Run benchmark tests"
        echo "  clean        Clean test artifacts and temporary files"
        echo "  all          Run all unit tests (default)"
        echo "  help         Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0                    # Run all unit tests"
        echo "  $0 unit              # Run only unit tests"
        echo "  $0 integration       # Run only integration tests"
        echo "  $0 coverage          # Run tests with coverage"
        echo ""
        ;;
    *)
        print_error "Unknown command: $1"
        print_status "Use '$0 help' to see available commands"
        exit 1
        ;;
esac