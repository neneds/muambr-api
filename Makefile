# Makefile for muambr-api test management

.PHONY: test test-unit test-integration test-extractors test-coverage test-bench clean-tests help

# Default target
test: test-unit

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	go test -race -v ./tests/unit/...

# Run integration tests (requires INTEGRATION_TESTS=true)
test-integration:
	@echo "Running integration tests..."
	INTEGRATION_TESTS=true go test -race -v ./tests/integration/...

# Run all extractor tests (unit + integration)
test-extractors:
	@echo "Running extractor tests..."
	go test -race -v ./tests/unit/extractors/...
	INTEGRATION_TESTS=true go test -race -v ./tests/integration/extractors/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p coverage
	go test -race -coverprofile=coverage/unit.out -covermode=atomic ./tests/unit/...
	go tool cover -html=coverage/unit.out -o coverage/unit.html
	@echo "Unit test coverage report: coverage/unit.html"
	
	INTEGRATION_TESTS=true go test -race -coverprofile=coverage/integration.out -covermode=atomic ./tests/integration/...
	go tool cover -html=coverage/integration.out -o coverage/integration.html
	@echo "Integration test coverage report: coverage/integration.html"

# Run benchmark tests
test-bench:
	@echo "Running benchmark tests..."
	go test -bench=. -benchmem ./tests/...

# Clean test artifacts
clean-tests:
	@echo "Cleaning test artifacts..."
	rm -rf coverage/
	find . -name "*.test" -delete
	find . -name "*.out" -delete
	@echo "Test artifacts cleaned"

# Run all tests (unit + integration)
test-all: test-unit test-integration

# Lint tests (requires golangci-lint)
lint-tests:
	@echo "Linting test files..."
	golangci-lint run ./tests/...

# Format test files
fmt-tests:
	@echo "Formatting test files..."
	go fmt ./tests/...

# Show test help
help:
	@echo "Available test commands:"
	@echo "  make test            - Run unit tests (default)"
	@echo "  make test-unit       - Run unit tests only"
	@echo "  make test-integration- Run integration tests only"
	@echo "  make test-extractors - Run all extractor tests"
	@echo "  make test-coverage   - Run tests with coverage reports"
	@echo "  make test-bench      - Run benchmark tests"
	@echo "  make test-all        - Run all tests"
	@echo "  make clean-tests     - Clean test artifacts"
	@echo "  make lint-tests      - Lint test files"
	@echo "  make fmt-tests       - Format test files"
	@echo "  make help            - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  INTEGRATION_TESTS=true  - Enable integration tests"