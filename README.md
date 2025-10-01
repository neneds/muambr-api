# Product Comparison API

A simplified Go REST API for product price comparisons across different countries and platforms.

## Project Structure

```
muambr-goapi/
├── main.go                    # Main application entry point
├── go.mod                     # Go module definition
├── models/                    # Data models
│   └── models.go
├── handlers/                  # HTTP handlers
│   └── comparison_handler.go
├── routes/                    # Route definitions
│   └── routes.go
└── extractors/                # Python scrapers
    ├── idealo_page.py
    └── kuantokusta_page.py
```

## API Endpoints

### GET /api/comparisons

Get product price comparisons from different platforms.

**Parameters:**
- `name` (required): Product name to search for
- `country` (required): Country ISO code for comparison target (PT, US, ES, DE, GB, BR)
- `currency` (optional): Base currency code (auto-detected from country if not provided)
- `currentCountry` (optional): User's current location ISO code for localized results

**Examples:**
```bash
# Basic comparison
curl "http://localhost:8080/api/comparisons?name=iPhone%2015&country=PT&currency=EUR"

# With current location context
curl "http://localhost:8080/api/comparisons?name=iPhone%2015&country=PT&currentCountry=US"
```

**Response:**
```json
[
  {
    "name": "Sony WH-1000XM6 - Auriculares Bluetooth con cancelación activa de ruido - Negro nuevo",
    "price": "342.87",
    "store": "Store audio",
    "currency": "EUR",
    "url": "https://www.idealo.es/relocator/relocate?categoryId=2520&offerKey=3755ad8b44312650d3c97c23bb8c93b1&offerListId=206509478-27E3E95F56F555BCADFDD6FB2FBB0E79&pos=3&price=342.87&productid=206509477&sid=335485&type=offer"
  },
  {
    "name": "iPhone 15 - Premium variant",
    "price": "289.99",
    "store": "Electronics Store Plus",
    "currency": "EUR",
    "url": "https://example.com/product2"
  }
]
```

**Supported Countries:**
- `PT` - Portugal (EUR)
- `US` - United States (USD)
- `ES` - Spain (EUR)
- `DE` - Germany (EUR)
- `GB` - United Kingdom (GBP)
- `BR` - Brazil (BRL)

### Health Check

- `GET /health` - API health check endpoint

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git

### Installation

1. Navigate to the goAPI directory:
   ```bash
   cd goAPI
   ```

2. Download dependencies:
   ```bash
   go mod tidy
   ```

3. Run the application:
   ```bash
   go run main.go
   ```

The API will start on `http://localhost:8080`

### Development

The current implementation provides the basic structure and endpoint definitions. The actual business logic for each endpoint needs to be implemented based on your requirements.

Each handler method includes TODO comments indicating where the actual implementation should be added.

## Models

The API includes Go models that correspond to the Swift models in your iOS application:

- `ProductEntry` - Main product entry model
- `ProductComparison` - Product comparison model
- `ProductPrice` - Price with currency information
- `SourceLink` - External source link
- `LocationData` - Geographic location data
- `Country` - Country enumeration
- `ProductCondition` - Product condition enumeration

## CORS

The API includes basic CORS middleware to allow cross-origin requests during development. Adjust the CORS settings as needed for your production environment.
