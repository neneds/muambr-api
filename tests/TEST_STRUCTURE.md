# Unit Test Structure for Go Extractors

This document outlines the comprehensive unit test structure implemented for the muambr-api Go extractors, organized similar to iOS project testing patterns.

## 📁 Test Structure Overview

```
tests/
├── unit/                           # Unit tests for individual components
│   ├── extractors/                # Extractor-specific unit tests
│   │   ├── registry_test.go       # ExtractorRegistry tests
│   │   ├── acharpromo_test.go     # AcharPromo extractor tests
│   │   ├── extractors_test.go     # Other extractor tests
│   │   └── acharpromo_testdata_test.go # Test data usage examples
│   ├── handlers/                  # Handler unit tests (future)
│   ├── models/                    # Model unit tests (future)
│   └── utils/                     # Utility function tests (future)
├── integration/                   # Integration tests
│   └── extractors/               # End-to-end extractor tests
│       └── extractors_integration_test.go
├── mocks/                        # Mock implementations
│   └── extractor_mock.go         # Mock extractor for testing
├── testdata/                     # Test fixtures and sample data
│   ├── html/                     # Sample HTML files
│   │   └── acharpromo_sample.html
│   └── json/                     # JSON response samples
│       └── acharpromo_sample_response.json
├── testhelpers/                  # Test utility functions
│   └── helpers.go                # Common test helpers
└── README.md                     # Test documentation
```

## 🧪 Test Categories

### 1. Unit Tests (`/tests/unit/`)

**What they test:**
- Individual extractor methods (GetCountryCode, GetIdentifier, etc.)
- Interface compliance
- Basic functionality without external dependencies

**Example:**
```go
func TestAcharPromoExtractor(t *testing.T) {
    extractor := extractors.NewAcharPromoExtractor()
    
    t.Run("GetCountryCode", func(t *testing.T) {
        country := extractor.GetCountryCode()
        expected := models.CountryBrazil
        if country != expected {
            t.Errorf("Expected country code %s, got %s", expected, country)
        }
    })
}
```

### 2. Integration Tests (`/tests/integration/`)

**What they test:**
- Real HTTP requests to external websites
- End-to-end extractor functionality
- Python script execution

**Requirements:**
- Set `INTEGRATION_TESTS=true` environment variable
- Internet connectivity
- Python environment with dependencies

### 3. Mock Tests (`/tests/mocks/`)

**What they provide:**
- Mock extractor implementations
- Controlled test scenarios
- Error simulation

**Usage:**
```go
mockExtractor := mocks.NewMockExtractor(models.CountryBrazil, "mock_extractor")
mockExtractor.WithResults(sampleResults)
```

## 🛠 Running Tests

### Using Make Commands
```bash
# Run unit tests only (fast, no external dependencies)
make test-unit

# Run integration tests (requires internet & Python)
make test-integration  

# Run all extractor tests
make test-extractors

# Run tests with coverage reports
make test-coverage

# Clean test artifacts
make clean-tests
```

### Using Shell Script
```bash
# Run unit tests
./run_tests.sh unit

# Run integration tests
./run_tests.sh integration

# Run with coverage
./run_tests.sh coverage

# Get help
./run_tests.sh help
```

### Using Go Commands Directly
```bash
# Unit tests only
go test -v ./tests/unit/...

# Integration tests (with environment variable)
INTEGRATION_TESTS=true go test -v ./tests/integration/...

# Specific extractor tests
go test -v ./tests/unit/extractors/

# With race detection and coverage
go test -race -cover ./tests/unit/...
```

## 📊 Test Features

### ✅ **Implemented Features**

1. **Structured Organization**: Tests organized in dedicated folders similar to iOS projects
2. **Mock System**: Comprehensive mock extractor for controlled testing
3. **Test Data**: Sample HTML and JSON files for realistic testing
4. **Test Helpers**: Utility functions for loading test data and assertions
5. **Integration Support**: Separate integration tests with proper environment checks
6. **Coverage Reports**: HTML coverage reports generation
7. **Multiple Run Options**: Make, shell script, and direct Go commands
8. **Race Detection**: Tests run with race condition detection
9. **Timeout Handling**: Integration tests with timeout protection
10. **Clean Documentation**: Clear instructions and examples

### 🔄 **Test Execution Flow**

1. **Unit Tests**: Fast execution, no external dependencies
2. **Mock Verification**: Test interface compliance and basic logic
3. **Test Data Loading**: Use realistic HTML/JSON samples
4. **Integration Tests**: Optional real-world testing with external sites
5. **Coverage Analysis**: Generate detailed coverage reports

### 📈 **Benefits of This Structure**

1. **Separation of Concerns**: Unit vs Integration vs Mock tests
2. **Fast Feedback**: Unit tests run quickly without network calls
3. **Realistic Testing**: Integration tests with real websites
4. **Easy Maintenance**: Clear organization and helper functions
5. **CI/CD Friendly**: Can run unit tests in CI, integration tests separately
6. **iOS-like Structure**: Familiar organization for iOS developers
7. **Comprehensive Coverage**: Both positive and negative test scenarios

## 🚀 **Adding New Tests**

### For a New Extractor:
1. Create `tests/unit/extractors/new_extractor_test.go`
2. Add integration test in `tests/integration/extractors/`
3. Create test data files in `tests/testdata/`
4. Use existing helpers and mock patterns

### For New Features:
1. Add unit tests for the feature logic
2. Create integration tests if external dependencies involved
3. Update test data as needed
4. Document any new test patterns

This structure provides a solid foundation for testing Go extractors while maintaining familiar iOS-like organization and comprehensive test coverage.