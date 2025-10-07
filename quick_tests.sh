#!/bin/bash

# Quick Test Commands for muambr-api
# Usage: source quick_tests.sh

# Set your base URL (change for production)
BASE_URL="http://localhost:8080"

echo "🧪 muambr-api Quick Test Commands"
echo "================================"
echo "Base URL: $BASE_URL"
echo ""

# Define test functions
health_check() {
    echo "🔍 Health Check:"
    curl -s "$BASE_URL/health" | jq '.'
    echo ""
}

test_brazil() {
    echo "🇧🇷 Testing Mercado Livre (Brazil):"
    curl -s "$BASE_URL/api/comparisons?name=sony%20xm6&country=BR&currency=BRL" | jq '.[:3]'
    echo ""
}

test_portugal() {
    echo "🇵🇹 Testing KuantoKusta (Portugal):"
    curl -s "$BASE_URL/api/comparisons?name=iphone%2015&country=PT&currency=EUR" | jq '.[:3]'
    echo ""
}

test_spain() {
    echo "🇪🇸 Testing Kelkoo (Spain):"
    curl -s "$BASE_URL/api/comparisons?name=samsung%20galaxy&country=ES&currency=EUR" | jq '.[:3]'
    echo ""
}

test_error() {
    echo "❌ Testing Error Response:"
    curl -s "$BASE_URL/api/comparisons?name=test&country=XX&currency=USD" | jq '.'
    echo ""
}

run_all_tests() {
    echo "🚀 Running all tests..."
    health_check
    test_brazil
    test_portugal  
    test_spain
    test_error
    echo "✅ All tests completed!"
}

# Individual curl commands for copy-paste

echo "📋 Individual Commands (copy & paste):"
echo ""

echo "# Health Check"
echo "curl -s '$BASE_URL/health' | jq '.'"
echo ""

echo "# Mercado Livre (Brazil) - Sony XM6"
echo "curl -s '$BASE_URL/api/comparisons?name=sony%20xm6&country=BR&currency=BRL' | jq '.'"
echo ""

echo "# KuantoKusta (Portugal) - iPhone 15" 
echo "curl -s '$BASE_URL/api/comparisons?name=iphone%2015&country=PT&currency=EUR' | jq '.'"
echo ""

echo "# Kelkoo (Spain) - Samsung Galaxy"
echo "curl -s '$BASE_URL/api/comparisons?name=samsung%20galaxy&country=ES&currency=EUR' | jq '.'"
echo ""

echo "# Error Test - Invalid Country"
echo "curl -s '$BASE_URL/api/comparisons?name=test&country=XX&currency=USD' | jq '.'"
echo ""

echo "📝 Available Functions:"
echo "  health_check    - Test health endpoint"
echo "  test_brazil     - Test Mercado Livre"
echo "  test_portugal   - Test KuantoKusta"
echo "  test_spain      - Test Kelkoo"
echo "  test_error      - Test error handling"
echo "  run_all_tests   - Run all tests"
echo ""

echo "Usage: health_check"
echo "       test_brazil"
echo "       run_all_tests"