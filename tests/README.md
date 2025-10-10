# Testing Documentation

This document describes the testing structure and setup for the muambr-goapi project, organized in a structure similar to iOS projects.

## Directory Structure

```
tests/
├── unit/                    # Unit tests for individual components
│   ├── extractors/         # Extractor-specific unit tests
│   ├── handlers/           # Handler unit tests
│   ├── models/             # Model unit tests
│   └── utils/              # Utility function tests
├── integration/            # Integration tests
│   ├── api/               # API endpoint integration tests
│   ├── extractors/        # End-to-end extractor tests
│   └── database/          # Database integration tests (if applicable)
├── mocks/                 # Mock implementations and test doubles
│   ├── extractors/        # Mock extractors
│   ├── http/              # Mock HTTP clients
│   └── services/          # Mock external services
└── testdata/              # Test fixtures and sample data
    ├── html/              # Sample HTML files for extractor testing
    ├── json/              # JSON response samples
    └── responses/         # HTTP response samples
```

## Running Tests

### Run all tests
```bash
go test ./tests/...
```

### Run only unit tests
```bash
go test ./tests/unit/...
```

### Run only integration tests
```bash
go test ./tests/integration/...
```

### Run specific extractor tests
```bash
go test ./tests/unit/extractors/...
```

### Run with coverage
```bash
go test -cover ./tests/...
```

### Run with verbose output
```bash
go test -v ./tests/...
```

## Test Conventions

1. **File Naming**: Test files should end with `_test.go`
2. **Function Naming**: Test functions should start with `Test`
3. **Benchmark Naming**: Benchmark functions should start with `Benchmark`
4. **Example Naming**: Example functions should start with `Example`
5. **Package Naming**: Test packages should match the package being tested with `_test` suffix for external tests

## Integration Test Environment

Integration tests may require:
- Internet connection for real HTTP requests
- Environment variables for API keys
- Python environment for extractor tests
- Mock servers for controlled testing