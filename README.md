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
│   ├── comparison_handler.go
│   └── extractor_handler.go
├── routes/                    # Route definitions
│   └── routes.go
└── extractors/                # Product extractors
    ├── extractor.go           # Extractor interface and registry
    ├── base_extractor.go      # Base extractor with gzip support
    ├── extractor.go           # Extractor interface
    └── acharpromo_extractor.go # Brazil (BR) - Pure Go implementation
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
# Basic comparison for Portugal
curl "http://localhost:8080/api/comparisons?name=iPhone%2015&country=PT&currency=EUR"

# Spain product search with Idealo
curl "http://localhost:8080/api/comparisons?name=sony%20wh-1000xm6&country=ES&currency=EUR"

# With current location context
curl "http://localhost:8080/api/comparisons?name=iPhone%2015&country=PT&currentCountry=US"
```

**Response:**
```json
[
  {
    "name": "Fone Sony Wh-1000xm6 Lançamento 2025",
    "price": "3997",
    "store": "Mercado Livre",
    "currency": "BRL",
    "url": "https://produto.mercadolivre.com.br/MLB-5450212900"
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

**Supported Extractors:**
- **KuantoKusta** (Portugal) - Portuguese price comparison service
  - URL Format: `https://www.kuantokusta.pt/search?q={product_name}`
- **Mercado Livre** (Brazil) - Brazilian marketplace and price comparison
  - URL Format: `https://lista.mercadolivre.com.br/{product-name}`
- **Idealo** (Spain) - Spanish price comparison and product search
  - URL Format: `https://www.idealo.es/resultados.html?q={product_name}`

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
