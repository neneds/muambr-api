# Copilot Instructions for muambr-goapi

## Project Overview
This is a pure Go REST API for multi-country product price comparisons. The system uses a pure Go architecture where Go handles HTTP routing, business logic, response processing, and web scraping from e-commerce sites using native Go libraries.

## Architecture Patterns

### Extractor Pattern
- All extractors implement the `Extractor` interface in `extractors/extractor.go`
- Each extractor targets specific countries/regions via `GetCountryCode()` and `GetMacroRegion()`
- Pure Go extractors use native HTTP clients with anti-bot protection and HTML parsing via goquery
- Extractors parse HTML directly and return `models.ProductComparison` objects

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
2. Implement pure Go HTML parsing with CSS selectors using goquery
3. Register extractor in `handlers/extractor_handler.go`
4. Add unit tests in `tests/unit/extractors/`
5. Add sample responses in `sample_responses/`

### Testing Commands
```bash
make test           # Unit tests only
make test-all       # Unit + integration tests
make test-coverage  # Generate HTML coverage reports
INTEGRATION_TESTS=true go test ./tests/integration/...
```

### Go Dependencies
- All dependencies managed via Go modules in `go.mod`
- HTML parsing handled by `github.com/PuerkitoBio/goquery`
- HTTP requests use standard library with anti-bot utilities

## Project-Specific Conventions

### Model Patterns
- Country codes use ISO format (`models.Country` enum: BR, PT, ES, etc.)
- Macro regions group countries (`models.MacroRegion`: EU, LATAM, NA)
- All prices include currency field, processed by `utils.ComparisonProcessor`

### Response Structure
- API returns `sections[]` grouped by country with `CountrySection` wrapper
- Each section contains `comparisons[]` of `ProductComparison` objects
- Price outlier filtering removes items >60% below average (configurable in `ComparisonProcessor`)

### HTTP Client Integration
- Go extractors use `utils.MakeScrapingRequest` for anti-bot protection
- HTML parsing done natively with goquery CSS selectors
- Gzip decompression handled automatically in base extractor

### Localization Keys
- API responses: `api.comparison.*`, `api.error.*`
- Health checks: `api.health.*`
- Country names: `countries.{CountryCode}`

## File Organization
- `handlers/` - HTTP request handlers with validation
- `extractors/` - Pure Go extractors with HTML parsing
- `linkparsers/` - HTML parsing for link preview feature (formerly htmlparser)
- `models/` - Domain models and enums
- `utils/` - Shared utilities (logging, comparison processing)
- `tests/unit/` vs `tests/integration/` - Separate test types with environment flag
- `sample_responses/` - Expected API response examples for reference

## Development Notes
- Use `INTEGRATION_TESTS=true` environment variable to enable tests that hit external APIs
- Coverage reports generated as HTML in `coverage/` directory
- Build process installs Go dependencies and compiles to single binary
- Gin router with CORS middleware configured for cross-origin requests

## Debugging and Code Cleanup Guidelines

### Debug Logging Best Practices
- **Always remove debug logs after troubleshooting** - Debug logs should never be committed to production code
- Use temporary debug logs during development with clear prefixes like `🔥`, `🔍`, `❌` for easy identification
- Common debug patterns to remove:
  - Method call logging: `utils.Info("🔥 ParserName.MethodName called")`
  - Step-by-step extraction debugging: `utils.Info("🔍 Found og:title", utils.String("title", title))`
  - Fallback usage warnings: `utils.Warn("🔍 Using fallback title")`
  - Parser type verification: `utils.String("parserType", fmt.Sprintf("%T", parser))`

### Post-Troubleshooting Cleanup Checklist
1. **Remove all debug log statements** added during troubleshooting session
2. **Remove unused imports** that were added for debugging (e.g., `fmt`, `utils` if only used for debug logs)
3. **Clean up temporary test files** (e.g., `debug_*.go` files)
4. **Verify functionality** still works after cleanup with a final test
5. **Keep only production-level logging** using appropriate log levels (Info/Warn/Error for meaningful events)

### HTML Parser Debugging Notes
- When debugging parser method overrides, verify that custom `ParseHTML` methods properly call overridden extraction methods
- Use temporary debug files (e.g., `debug_parser.go`) for isolated testing, then remove them
- Parser registry selection can be verified with minimal logging, but remove detailed parser type information
- Method resolution issues often require custom `ParseHTML` implementation to ensure proper method override behavior