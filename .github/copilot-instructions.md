# Copilot Instructions for muambr-goapi

## Project Overview
This is a Go REST API for multi-country product price comparisons. The system uses a hybrid Go + Python architecture where Go handles HTTP routing, business logic, and response processing, while Python scripts perform the actual web scraping from e-commerce sites.

## Architecture Patterns

### Extractor Pattern
- All extractors implement the `Extractor` interface in `extractors/extractor.go`
- Each extractor targets specific countries/regions via `GetCountryCode()` and `GetMacroRegion()`
- Go extractors shell out to Python scripts in `extractors/pythonExtractors/` using `exec.Command`
- Python scripts return JSON to stdout, which Go parses into `models.ProductComparison`

### Registry-Based Service Discovery
- `ExtractorRegistry` manages extractor instances by country/region
- Register extractors in `routes/routes.go` via `registry.RegisterExtractor()`
- Handlers query registry for available extractors by country or macro region

### Localized Error Handling
- All user-facing strings use `localization.T("key")` from JSON files in `localization/`
- Error responses include localized messages for i18n support
- Use `utils.Info/Warn/Error()` for structured logging with zap

## Key Workflows

### Adding New Extractors
1. Create Go extractor in `extractors/` implementing `Extractor` interface
2. Add corresponding Python scraper in `extractors/pythonExtractors/`
3. Register extractor in `routes/routes.go`
4. Add unit tests in `tests/unit/extractors/`
5. Add sample responses in `sample_responses/`

### Testing Commands
```bash
make test           # Unit tests only
make test-all       # Unit + integration tests
make test-coverage  # Generate HTML coverage reports
INTEGRATION_TESTS=true go test ./tests/integration/...
```

### Python Dependencies
- Python packages are installed locally to `./python_packages/` via `build.sh`
- Use `pip3 install -r requirements.txt --target ./python_packages` for deployment
- Python scripts expect this local package directory in their import paths

## Project-Specific Conventions

### Model Patterns
- Country codes use ISO format (`models.Country` enum: BR, PT, ES, etc.)
- Macro regions group countries (`models.MacroRegion`: EU, LATAM, NA)
- All prices include currency field, processed by `utils.ComparisonProcessor`

### Response Structure
- API returns `sections[]` grouped by country with `CountrySection` wrapper
- Each section contains `comparisons[]` of `ProductComparison` objects
- Price outlier filtering removes items >60% below average (configurable in `ComparisonProcessor`)

### Python Integration
- Go extractors use `exec.Command` to call Python scripts with product names as args
- Python scripts output JSON to stdout, errors to stderr
- Always handle both stdout/stderr and check exit codes when calling Python

### Localization Keys
- API responses: `api.comparison.*`, `api.error.*`
- Health checks: `api.health.*`
- Country names: `countries.{CountryCode}`

## File Organization
- `handlers/` - HTTP request handlers with validation
- `extractors/` - Go extractors + Python scrapers subdirectory
- `htmlparser/` - HTML parsing for link preview feature
- `models/` - Domain models and enums
- `utils/` - Shared utilities (logging, comparison processing)
- `tests/unit/` vs `tests/integration/` - Separate test types with environment flag
- `sample_responses/` - Expected API response examples for reference

## Development Notes
- Use `INTEGRATION_TESTS=true` environment variable to enable tests that hit external APIs
- Coverage reports generated as HTML in `coverage/` directory
- Build process installs Python deps locally for containerized deployment
- Gin router with CORS middleware configured for cross-origin requests