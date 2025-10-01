#!/bin/bash

# Test script for Muambr Go API

echo "Starting Muambr API server..."
cd /Users/dennismerli/Documents/Projects/muambr/goAPI

# Start the server in background
go run main.go &
SERVER_PID=$!

# Wait for server to start
sleep 3

echo "Testing API endpoints..."

# Test health endpoint
echo "1. Testing health endpoint:"
curl -s http://localhost:8080/health | jq '.'

echo -e "\n2. Testing product entries endpoint with pagination:"
curl -s "http://localhost:8080/api/product-entries?page=1&numberOfItems=5" | jq '.'

echo -e "\n3. Testing price conversion endpoint:"
curl -s -X POST http://localhost:8080/api/convert-price \
  -H "Content-Type: application/json" \
  -d '{
    "productPrice": {
      "uuid": "test-uuid",
      "currencyCode": "USD",
      "price": 100.0
    },
    "toCurrency": "EUR"
  }' | jq '.'

# Stop the server
echo -e "\nStopping server..."
kill $SERVER_PID

echo "Test completed!"
