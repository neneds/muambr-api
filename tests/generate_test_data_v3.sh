#!/bin/bash

# HTML Test Data Generator Script (Version 3 - With Brotli Support)
# This script fetches real HTML content from all extractor websites for testing purposes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SEARCH_TERM="iPad 10"
OUTPUT_DIR="/Users/dennismerli/Documents/Projects/muambr-goapi/tests/testdata/html"
ENCODED_SEARCH="iPad%2010"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Check for required tools
check_dependencies() {
    local missing_tools=()
    
    if ! command -v curl &> /dev/null; then
        missing_tools+=("curl")
    fi
    
    if ! command -v brotli &> /dev/null; then
        print_warning "brotli tool not found. Installing via Homebrew..."
        if command -v brew &> /dev/null; then
            brew install brotli
        else
            missing_tools+=("brotli")
        fi
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        print_error "Missing required tools: ${missing_tools[*]}"
        print_error "Please install them and run the script again"
        exit 1
    fi
}

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

print_status "Starting HTML test data collection for: $SEARCH_TERM"
print_status "Output directory: $OUTPUT_DIR"
print_status "Checking dependencies..."
check_dependencies
echo

# Function to fetch HTML with proper headers and error handling
fetch_html() {
    local name=$1
    local url=$2
    local output_file=$3
    
    print_step "Fetching HTML from $name..."
    print_status "URL: $url"
    
    # Create temporary file for the response
    local temp_file=$(mktemp)
    local temp_headers=$(mktemp)
    
    # Use curl with proper headers to mimic a real browser
    local http_status
    http_status=$(curl -s -o "$temp_file" -w "%{http_code}" \
        -D "$temp_headers" \
        -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36" \
        -H "Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8" \
        -H "Accept-Language: pt-BR,pt;q=0.9,en;q=0.8" \
        -H "Accept-Encoding: gzip, deflate, br" \
        -H "Connection: keep-alive" \
        -H "Upgrade-Insecure-Requests: 1" \
        -H "Cache-Control: no-cache" \
        -L \
        --max-time 30 \
        --connect-timeout 10 \
        "$url" 2>/dev/null || echo "000")
    
    if [[ "$http_status" == "200" ]]; then
        # Check if response is compressed
        local content_encoding=$(grep -i "content-encoding:" "$temp_headers" | cut -d: -f2 | tr -d ' \r\n' || echo "")
        local original_size=$(wc -c < "$temp_file")
        
        # Decompress based on content encoding
        case "$content_encoding" in
            "gzip")
                print_status "  Decompressing gzipped content..."
                if gunzip -c "$temp_file" > "${temp_file}.decompressed" 2>/dev/null; then
                    mv "${temp_file}.decompressed" "$temp_file"
                    print_status "  ✓ Gzip decompression successful"
                else
                    print_warning "  Failed to decompress gzip, using raw content"
                fi
                ;;
            "br")
                print_status "  Decompressing Brotli content..."
                if brotli -d "$temp_file" -o "${temp_file}.decompressed" 2>/dev/null; then
                    mv "${temp_file}.decompressed" "$temp_file"
                    print_status "  ✓ Brotli decompression successful"
                else
                    print_warning "  Failed to decompress Brotli, using raw content"
                fi
                ;;
            "deflate")
                print_status "  Decompressing deflate content..."
                # Use python for deflate decompression as it's more reliable
                if command -v python3 &> /dev/null; then
                    python3 -c "
import zlib
import sys
with open('$temp_file', 'rb') as f:
    data = f.read()
try:
    decompressed = zlib.decompress(data)
    with open('${temp_file}.decompressed', 'wb') as f:
        f.write(decompressed)
    print('Deflate decompression successful')
except:
    print('Deflate decompression failed')
    sys.exit(1)
" && mv "${temp_file}.decompressed" "$temp_file" || print_warning "  Failed to decompress deflate, using raw content"
                else
                    print_warning "  Python3 not available for deflate decompression, using raw content"
                fi
                ;;
            *)
                if [[ -n "$content_encoding" ]]; then
                    print_status "  Content encoding: $content_encoding (no decompression needed)"
                else
                    print_status "  No content encoding detected"
                fi
                ;;
        esac
        
        # Add metadata header to the file
        cat > "$output_file" << EOF
<!-- 
Test Data Generated: $(date)
Original URL: $url
Search Term: $SEARCH_TERM
HTTP Status: $http_status
Content Encoding: ${content_encoding:-none}
Original Size: $original_size bytes
Final Size: $(wc -c < "$temp_file") bytes
-->
EOF
        
        # Append the HTML content
        cat "$temp_file" >> "$output_file"
        
        local final_size=$(wc -c < "$output_file")
        print_status "✓ Success: Saved $final_size bytes to $(basename "$output_file")"
        
        # Quick validation - check if it looks like HTML
        if head -20 "$output_file" | grep -qi "<!DOCTYPE\|<html\|<head\|<body"; then
            print_status "  ✓ Content appears to be valid HTML"
        else
            print_warning "  ⚠ Content might not be standard HTML (could be API response or error page)"
            # Show first few lines for debugging
            print_status "  First few lines of content:"
            head -5 "$temp_file" | sed 's/^/    /'
        fi
        
    else
        print_error "✗ Failed: HTTP $http_status"
        
        # Save error response for debugging if we got some content
        if [[ -s "$temp_file" ]]; then
            local error_file="${output_file%.html}_error_${http_status}.html"
            cat > "$error_file" << EOF
<!-- 
Error Response Generated: $(date)
Original URL: $url
Search Term: $SEARCH_TERM
HTTP Status: $http_status
-->
EOF
            cat "$temp_file" >> "$error_file"
            print_status "  Saved error response to $(basename "$error_file")"
        fi
    fi
    
    # Cleanup
    rm -f "$temp_file" "$temp_headers"
    echo
}

# Define extractors with their search URLs
print_status "Configured extractors:"

# AcharPromo (Brazil) 
ACHARPROMO_URL="https://achar.promo/search?q=${ENCODED_SEARCH}"
print_status "  • AcharPromo: $ACHARPROMO_URL"

# MercadoLivre (Brazil)
MERCADOLIVRE_URL="https://lista.mercadolivre.com.br/${ENCODED_SEARCH}"
print_status "  • MercadoLivre: $MERCADOLIVRE_URL"

# KuantoKusta (Portugal)
KUANTOKUSTA_URL="https://www.kuantokusta.pt/search?q=${ENCODED_SEARCH}"
print_status "  • KuantoKusta: $KUANTOKUSTA_URL"

# Kelkoo (Spain)
KELKOO_URL="https://www.kelkoo.es/ss-${ENCODED_SEARCH}.html"
print_status "  • Kelkoo: $KELKOO_URL"

echo

# Fetch HTML from all extractors
fetch_html "AcharPromo" "$ACHARPROMO_URL" "$OUTPUT_DIR/acharpromo_ipad10_search.html"
fetch_html "MercadoLivre" "$MERCADOLIVRE_URL" "$OUTPUT_DIR/mercadolivre_ipad10_search.html"
fetch_html "KuantoKusta" "$KUANTOKUSTA_URL" "$OUTPUT_DIR/kuantokusta_ipad10_search.html"
fetch_html "Kelkoo" "$KELKOO_URL" "$OUTPUT_DIR/kelkoo_ipad10_search.html"

# Summary
echo
print_status "HTML Test Data Collection Complete!"
print_status "Files generated in: $OUTPUT_DIR"
echo

print_step "Summary of collected files:"
if ls -la "$OUTPUT_DIR"/*.html 2>/dev/null; then
    echo
    print_status "You can now use these HTML files in your unit tests to test the extractors with real data."
else
    print_warning "No HTML files were generated successfully."
fi

echo
print_status "Script completed at $(date)"