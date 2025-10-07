# Environment Configuration

## Local Development Setup

### 1. Configure Exchange Rate API

1. **Get a free API key** from [exchangerate-api.com](https://exchangerate-api.com/)
   - Sign up for a free account (1,500 requests/month)
   - Copy your API key

2. **Configure local environment**:
   ```bash
   # Copy the example file
   cp .env.example .env
   
   # Edit .env and add your API key
   # Replace 'your_api_key_here' with your actual API key
   EXCHANGE_RATE_API_KEY=abcd1234-your-actual-api-key-here
   ```

3. **Test the configuration**:
   ```bash
   # Start the server
   go run main.go
   
   # Test with real exchange rates
   curl "http://localhost:8080/admin/exchange-rates/test?currency=USD"
   ```

### 2. Environment Variables

The application supports these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `EXCHANGE_RATE_API_KEY` | _(empty)_ | Your exchangerate-api.com API key |
| `PORT` | `8080` | Server port |
| `GIN_MODE` | `debug` | Gin mode: `debug`, `release`, or `test` |

### 3. Production Deployment (Render.com)

In your Render.com dashboard:

1. Go to your service settings
2. Navigate to "Environment" section  
3. Add environment variable:
   - **Key**: `EXCHANGE_RATE_API_KEY`
   - **Value**: Your actual API key

### 4. Testing Different Configurations

#### Without API Key (Mock Data)
```bash
# Remove or leave empty the API key
EXCHANGE_RATE_API_KEY=

# Start server - will use mock exchange rates
go run main.go
```

#### With API Key (Real Data)
```bash
# Set your API key
EXCHANGE_RATE_API_KEY=your_actual_key

# Start server - will use real exchange rates  
go run main.go
```

### 5. Monitoring Exchange Rates

Use the admin endpoints to monitor the system:

```bash
# Check cache status
curl "http://localhost:8080/admin/exchange-rates/status"

# Test API with specific currency
curl "http://localhost:8080/admin/exchange-rates/test?currency=EUR"

# Clear cache (for testing)
curl -X DELETE "http://localhost:8080/admin/exchange-rates/cache"
```

### 6. Example .env File

```env
# Exchange Rate API Configuration
EXCHANGE_RATE_API_KEY=abc123def456-your-key-here

# Server Configuration  
PORT=8080
GIN_MODE=debug
```

## Notes

- The `.env` file is ignored by Git for security
- Always use `.env.example` as a template
- In production, set environment variables directly in your hosting platform
- The app gracefully falls back to mock data if the API key is invalid or missing