#!/bin/bash

# API Test Suite for muambr-api
# Usage: ./test_api.sh [base_url]

BASE_URL=${1:-"http://localhost:8080"}
TOTAL_TESTS=0
PASSED_TESTS=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧪 muambr-api Test Suite${NC}"
echo -e "${BLUE}========================${NC}"
echo "Base URL: $BASE_URL"
echo ""

test_endpoint() {
    local name="$1"
    local url="$2"
    local expected_status="$3"
    local test_response_body="$4"
    
    echo -n -e "Testing ${YELLOW}$name${NC}... "
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    # Make the request and capture both status and response
    response=$(curl -s -w "\n%{http_code}" "$url")
    status=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$status" = "$expected_status" ]; then
        if [ "$test_response_body" = "true" ] && [ -n "$body" ]; then
            # Test if response is valid JSON
            if echo "$body" | jq . >/dev/null 2>&1; then
                echo -e "${GREEN}✅ PASS${NC} (HTTP $status, Valid JSON)"
                PASSED_TESTS=$((PASSED_TESTS + 1))
            else
                echo -e "${RED}❌ FAIL${NC} (HTTP $status, Invalid JSON)"
            fi
        else
            echo -e "${GREEN}✅ PASS${NC} (HTTP $status)"
            PASSED_TESTS=$((PASSED_TESTS + 1))
        fi
    else
        echo -e "${RED}❌ FAIL${NC} (HTTP $status, expected $expected_status)"
        echo -e "   Response: $body"
    fi
}

test_product_search() {
    local name="$1"
    local product="$2"
    local country="$3"
    local currency="$4"
    
    local encoded_product=$(echo "$product" | sed 's/ /%20/g')
    local url="$BASE_URL/api/comparisons?name=$encoded_product&country=$country&currency=$currency"
    
    echo -n -e "Testing ${YELLOW}$name${NC} ($product)... "
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    response=$(curl -s -w "\n%{http_code}" "$url")
    status=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$status" = "200" ]; then
        if echo "$body" | jq . >/dev/null 2>&1; then
            product_count=$(echo "$body" | jq 'length')
            if [ "$product_count" -gt 0 ]; then
                echo -e "${GREEN}✅ PASS${NC} (HTTP $status, $product_count products found)"
                PASSED_TESTS=$((PASSED_TESTS + 1))
            else
                echo -e "${YELLOW}⚠️  PARTIAL${NC} (HTTP $status, No products found)"
                PASSED_TESTS=$((PASSED_TESTS + 1))
            fi
        else
            echo -e "${RED}❌ FAIL${NC} (HTTP $status, Invalid JSON)"
        fi
    else
        echo -e "${RED}❌ FAIL${NC} (HTTP $status)"
        echo -e "   Response: $body"
    fi
}

# Check if server is running
echo -n "Checking if server is accessible... "
if curl -s --connect-timeout 5 "$BASE_URL/health" >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Server is running${NC}"
else
    echo -e "${RED}❌ Server is not accessible${NC}"
    echo "Make sure the server is running on $BASE_URL"
    exit 1
fi

echo ""

# Basic endpoint tests
echo -e "${BLUE}📋 Basic Endpoint Tests${NC}"
test_endpoint "Health Check" "$BASE_URL/health" "200" "true"

echo ""

# Product search tests
echo -e "${BLUE}🔍 Product Search Tests${NC}"
test_product_search "Mercado Livre (BR)" "sony xm6" "BR" "BRL"
test_product_search "KuantoKusta (PT)" "iphone 15" "PT" "EUR"
test_product_search "Kelkoo (ES)" "samsung galaxy" "ES" "EUR"

echo ""

# Error handling tests
echo -e "${BLUE}❌ Error Handling Tests${NC}"
test_endpoint "Invalid Country Code" "$BASE_URL/api/comparisons?name=test&country=XX&currency=USD" "400"
test_endpoint "Missing Product Name" "$BASE_URL/api/comparisons?country=BR&currency=BRL" "400"
test_endpoint "Missing Country" "$BASE_URL/api/comparisons?name=test&currency=BRL" "400"

echo ""

# Advanced tests
echo -e "${BLUE}🚀 Advanced Tests${NC}"
test_product_search "Special Characters" "café expresso" "PT" "EUR"
test_product_search "Long Product Name" "apple macbook pro 16 inch 2024" "BR" "BRL"
test_product_search "Numbers in Name" "iphone 15 pro max 256gb" "ES" "EUR"

echo ""

# Performance test
echo -e "${BLUE}⚡ Performance Test${NC}"
echo -n "Measuring response time... "
response_time=$(curl -o /dev/null -s -w "%{time_total}" "$BASE_URL/health")
echo -e "${YELLOW}${response_time}s${NC}"

echo ""

# Results summary
echo -e "${BLUE}📊 Test Results${NC}"
echo "========================"
if [ $PASSED_TESTS -eq $TOTAL_TESTS ]; then
    echo -e "${GREEN}🎉 All tests passed! ($PASSED_TESTS/$TOTAL_TESTS)${NC}"
    exit 0
else
    failed_tests=$((TOTAL_TESTS - PASSED_TESTS))
    echo -e "${RED}💥 $failed_tests test(s) failed! ($PASSED_TESTS/$TOTAL_TESTS passed)${NC}"
    exit 1
fi