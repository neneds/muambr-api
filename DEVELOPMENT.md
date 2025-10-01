# Development Guide

## API Endpoints

This Go API implements all the endpoints defined in the Swift `MuambrAPIClient` protocol:

### Product Entries

| Method | Endpoint | Description | Status |
|--------|----------|-------------|---------|
| GET | `/api/product-entries?page=X&numberOfItems=Y` | Fetch product entries with pagination | ✅ Implemented |
| GET | `/api/product-entries/:uuid` | Fetch specific product entry | 🚧 Placeholder |
| POST | `/api/product-entries` | Create new product entry | 🚧 Placeholder |
| PUT | `/api/product-entries/:uuid` | Update existing product entry | 🚧 Placeholder |
| DELETE | `/api/product-entries/:uuid` | Delete product entry | 🚧 Placeholder |

### Product Comparisons

| Method | Endpoint | Description | Status |
|--------|----------|-------------|---------|
| GET | `/api/product-entries/:uuid/comparisons` | Fetch product comparisons | 🚧 Placeholder |
| POST | `/api/product-entries/:uuid/comparisons` | Create product comparisons | 🚧 Placeholder |

### Favorites

| Method | Endpoint | Description | Status |
|--------|----------|-------------|---------|
| PUT | `/api/product-entries/:uuid/favorite` | Update favorite status | 🚧 Placeholder |

### Price Conversion

| Method | Endpoint | Description | Status |
|--------|----------|-------------|---------|
| POST | `/api/convert-price` | Convert product price to another currency | ✅ Basic implementation |

### Health Check

| Method | Endpoint | Description | Status |
|--------|----------|-------------|---------|
| GET | `/health` | API health check | ✅ Implemented |

## Development Tasks

### Phase 1: Database Integration
- [ ] Add database connection (PostgreSQL/MySQL recommended)
- [ ] Create database schemas matching the Go models
- [ ] Implement database migrations
- [ ] Add database connection pooling

### Phase 2: Business Logic Implementation
- [ ] Implement `FetchProductEntry` - retrieve single product entry by UUID
- [ ] Implement `SaveProductEntry` - create new product entries
- [ ] Implement `UpdateProductEntry` - update existing product entries
- [ ] Implement `DeleteProductEntry` - soft/hard delete product entries
- [ ] Implement `CreateProductComparisons` - create comparisons for products
- [ ] Implement `FetchProductComparisons` - retrieve comparisons for a product
- [ ] Implement `SaveProductEntryFavorite` - toggle favorite status
- [ ] Enhance `GetConvertedPrice` - integrate with real currency conversion API

### Phase 3: External Integrations
- [ ] Integrate with currency conversion API (e.g., exchangerate-api.com)
- [ ] Add image upload and storage for product images
- [ ] Implement location services integration

### Phase 4: Authentication & Security
- [ ] Add user authentication (JWT recommended)
- [ ] Implement authorization middleware
- [ ] Add request validation
- [ ] Add rate limiting
- [ ] Implement CORS properly for production

### Phase 5: Performance & Monitoring
- [ ] Add logging (structured logging with logrus/zap)
- [ ] Add metrics collection
- [ ] Implement caching (Redis recommended)
- [ ] Add database query optimization
- [ ] Add API documentation with Swagger

## File Structure

```
goAPI/
├── main.go                           # Application entry point
├── go.mod                           # Go module definition
├── go.sum                           # Dependency checksums
├── README.md                        # Project documentation
├── test-api.sh                      # API testing script
├── DEVELOPMENT.md                   # This file
├── models/
│   └── models.go                    # Data models
├── handlers/
│   └── product_entry_handler.go     # HTTP request handlers
├── routes/
│   └── routes.go                    # Route definitions
├── database/                        # (To be created)
│   ├── connection.go               # Database connection
│   ├── migrations/                 # Database migrations
│   └── repositories/               # Data access layer
├── services/                        # (To be created)
│   ├── product_service.go          # Business logic
│   ├── comparison_service.go       # Comparison logic
│   └── currency_service.go         # Currency conversion
├── middleware/                      # (To be created)
│   ├── auth.go                     # Authentication middleware
│   ├── cors.go                     # CORS middleware
│   └── logging.go                  # Logging middleware
└── config/                          # (To be created)
    └── config.go                   # Configuration management
```

## Testing

Run the test script to verify all endpoints:

```bash
./test-api.sh
```

## Running the Server

```bash
# Development mode
go run main.go

# Build and run
go build .
./muambr-api
```

The server will start on `http://localhost:8080`
