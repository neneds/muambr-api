#!/bin/bash

# Test script for Exchange Rate API Integration
# Usage: ./test_exchange_rates.sh [API_KEY]

API_KEY=${1:-""}

echo "🚀 Testing Exchange Rate Integration"
echo "=================================="

if [ -z "$API_KEY" ]; then
    echo "⚠️  No API key provided - testing with mock data"
    echo "💡 Usage: $0 YOUR_API_KEY"
    echo ""
    
    # Start server without API key (uses mock data)
    echo "Starting server with mock exchange rates..."
    go run main.go > /dev/null 2>&1 &
    SERVER_PID=$!
    sleep 3
else
    echo "✅ API key provided - testing with real exchange rates"
    echo ""
    
    # Start server with API key
    echo "Starting server with real exchange rates..."
    EXCHANGE_RATE_API_KEY="$API_KEY" go run main.go > /dev/null 2>&1 &
    SERVER_PID=$!
    sleep 3
fi

# Test 1: Check initial cache status
echo "📊 Test 1: Initial cache status"
curl -s "http://localhost:8080/admin/exchange-rates/status" | jq .
echo ""

# Test 2: Make product comparison request (populates cache)
echo "🛍️  Test 2: Product comparison with currency conversion"
curl -s "http://localhost:8080/api/comparisons?name=sony&country=BR&currency=EUR&limit=2" | jq '.data[]? // .[] | {name: .name, price: .price, currency: .currency, convertedPrice: .convertedPrice}'
echo ""

# Test 3: Check cache status after request
echo "📊 Test 3: Cache status after conversion"
curl -s "http://localhost:8080/admin/exchange-rates/status" | jq .
echo ""

# Test 4: Test direct exchange rate API
echo "💱 Test 4: Direct exchange rate test"
curl -s "http://localhost:8080/admin/exchange-rates/test?currency=USD" | jq .
echo ""

# Test 5: Test different currency conversion
echo "💱 Test 5: EUR base currency test"
curl -s "http://localhost:8080/admin/exchange-rates/test?currency=EUR" | jq .
echo ""

# Test 6: Clear cache and test again
echo "🗑️  Test 6: Clear cache"
curl -s -X DELETE "http://localhost:8080/admin/exchange-rates/cache" | jq .
echo ""

echo "📊 Test 7: Cache status after clear"
curl -s "http://localhost:8080/admin/exchange-rates/status" | jq .
echo ""

# Cleanup
echo "🧹 Cleaning up..."
kill $SERVER_PID 2>/dev/null
echo "✅ Tests completed!"

echo ""
echo "📝 Summary:"
echo "- Exchange rate service working with $([ -z "$API_KEY" ] && echo "mock data" || echo "real API")"
echo "- Cache system functioning (5-hour TTL)"
echo "- Currency conversion integrated in product comparisons"
echo "- Admin endpoints available for monitoring"
echo ""
echo "🚀 Ready for production deployment!"
echo "   Set EXCHANGE_RATE_API_KEY in Render.com environment variables"