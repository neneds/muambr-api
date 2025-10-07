# API Testing - Sample Requests

## Health Check

### Basic Health Check
```bash
curl -X GET "http://localhost:8080/health"
```

**Expected Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-10-06T21:40:53Z"
}
```

## Product Comparisons

### 1. Mercado Livre (Brazil) - Sony Headphones
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=sony%20xm6&country=BR&currency=BRL"
```

### 2. KuantoKusta (Portugal) - iPhone
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=iphone%2015&country=PT&currency=EUR"
```

### 3. Kelkoo (Spain) - Samsung Galaxy
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=samsung%20galaxy%20s24&country=ES&currency=EUR"
```

### 4. With Current Country Context
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=macbook%20pro&country=PT&currentCountry=BR&currency=EUR"
```

### 5. Macro Region Search (All Americas)
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=nintendo%20switch&country=BR&currentCountry=US&currency=BRL"
```

## Formatted Requests (Pretty Print)

### Mercado Livre with JSON formatting
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=sony%20xm6&country=BR&currency=BRL" | jq '.'
```

### Multiple Products Test
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=playstation%205&country=PT&currency=EUR" | jq '.'
```

## Error Cases

### Invalid Country Code
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=test&country=XX&currency=USD"
```

### Missing Required Parameters
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=test"
```

### Empty Product Name
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=&country=BR&currency=BRL"
```

## Advanced Tests

### Special Characters in Product Name
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=caf%C3%A9%20expresso&country=PT&currency=EUR"
```

### Long Product Name
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=apple%20macbook%20pro%2016%20inch%202024%20m4%20chip&country=BR&currency=BRL"
```

### Numbers in Product Name
```bash
curl -X GET "http://localhost:8080/api/comparisons?name=iphone%2015%20pro%20max%20256gb&country=ES&currency=EUR"
```

## Production URLs (Replace with your Render URL)

### Health Check (Production)
```bash
curl -X GET "https://your-app-name.onrender.com/health"
```

### Product Search (Production)
```bash
curl -X GET "https://your-app-name.onrender.com/api/comparisons?name=sony%20xm6&country=BR&currency=BRL"
```

## Response Examples

### Successful Response
```json
[
  {
    "name": "Sony WH-1000XM6 Wireless Headphones",
    "price": "3997.00",
    "store": "Mercado Livre (Available in Brazil)",
    "currency": "BRL",
    "url": "https://produto.mercadolivre.com.br/MLB-5450212900-..."
  },
  {
    "name": "Fones De Ouvido Sem Fio Sony Wh-1000xm5s",
    "price": "2538.69", 
    "store": "Mercado Livre (Available in Brazil)",
    "currency": "BRL",
    "url": "https://produto.mercadolivre.com.br/MLB-3821476129-..."
  }
]
```

### Error Response
```json
{
  "error": "Invalid country ISO code: XX",
  "code": "XX",
  "supportedCodes": ["PT", "US", "ES", "DE", "GB", "BR"]
}
```

## Testing with Different HTTP Clients

### Using HTTPie
```bash
http GET localhost:8080/api/comparisons name=="sony xm6" country==BR currency==BRL
```

### Using wget
```bash
wget -qO- "http://localhost:8080/api/comparisons?name=sony%20xm6&country=BR&currency=BRL"
```

### Using PowerShell (Windows)
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/comparisons?name=sony%20xm6&country=BR&currency=BRL"
```

## Load Testing

### Simple Load Test (100 requests)
```bash
for i in {1..100}; do
  curl -s "http://localhost:8080/health" > /dev/null &
done
wait
```

### Sequential Product Searches
```bash
products=("sony xm6" "iphone 15" "samsung galaxy" "macbook pro" "nintendo switch")
for product in "${products[@]}"; do
  echo "Testing: $product"
  curl -s "http://localhost:8080/api/comparisons?name=${product// /%20}&country=BR&currency=BRL" | jq '.[] | .name' | head -3
  echo "---"
done
```

## Monitoring & Debugging

### Check Response Headers
```bash
curl -I "http://localhost:8080/health"
```

### Measure Response Time
```bash
curl -o /dev/null -s -w "Total time: %{time_total}s\n" "http://localhost:8080/api/comparisons?name=sony%20xm6&country=BR&currency=BRL"
```

### Verbose Output for Debugging
```bash
curl -v "http://localhost:8080/api/comparisons?name=test&country=BR&currency=BRL"
```

## Browser Testing

### Open in Browser
```
http://localhost:8080/health
http://localhost:8080/api/comparisons?name=sony%20xm6&country=BR&currency=BRL
```

## API Testing Tools

### Postman Collection Variables
- **base_url**: `http://localhost:8080` (local) or `https://your-app.onrender.com` (production)
- **product_name**: `sony xm6`
- **country**: `BR`
- **currency**: `BRL`

### Example Postman Request
```
GET {{base_url}}/api/comparisons?name={{product_name}}&country={{country}}&currency={{currency}}
```

## Automated Testing Script

### Bash Test Suite
```bash
#!/bin/bash

BASE_URL="http://localhost:8080"
TOTAL_TESTS=0
PASSED_TESTS=0

test_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="$3"
    
    echo -n "Testing $name... "
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    status=$(curl -s -o /dev/null -w "%{http_code}" "$url")
    
    if [ "$status" = "$expected_status" ]; then
        echo "✅ PASS (HTTP $status)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "❌ FAIL (HTTP $status, expected $expected_status)"
    fi
}

echo "🧪 Running API Tests..."
echo "========================"

# Health check
test_endpoint "Health Check" "$BASE_URL/health" "200"

# Valid requests
test_endpoint "Mercado Livre Search" "$BASE_URL/api/comparisons?name=sony%20xm6&country=BR&currency=BRL" "200"
test_endpoint "KuantoKusta Search" "$BASE_URL/api/comparisons?name=iphone&country=PT&currency=EUR" "200"
test_endpoint "Kelkoo Search" "$BASE_URL/api/comparisons?name=samsung&country=ES&currency=EUR" "200"

# Error cases
test_endpoint "Invalid Country" "$BASE_URL/api/comparisons?name=test&country=XX&currency=USD" "400"
test_endpoint "Missing Parameters" "$BASE_URL/api/comparisons?name=test" "400"

echo "========================"
echo "📊 Results: $PASSED_TESTS/$TOTAL_TESTS tests passed"

if [ $PASSED_TESTS -eq $TOTAL_TESTS ]; then
    echo "🎉 All tests passed!"
    exit 0
else
    echo "💥 Some tests failed!"
    exit 1
fi
```

## Performance Benchmarks

### Response Time Expectations
- **Health Check**: < 50ms
- **Product Search**: 2-10 seconds (depends on web scraping)
- **Error Responses**: < 100ms

### Concurrent Users Test
```bash
# Test with 10 concurrent users
seq 10 | xargs -n1 -P10 bash -c 'curl -s "http://localhost:8080/health" > /dev/null'
```

---

Save these requests and use them to thoroughly test your API before and after deployment! 🧪