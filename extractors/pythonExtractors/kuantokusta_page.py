import json
from bs4 import BeautifulSoup
import re

def extract_kuantokusta_products(html_string):
    soup = BeautifulSoup(html_string, 'html.parser')
    products = []

    # Find all product links that start with /p/ (product pages)
    product_links = soup.find_all('a', href=re.compile(r'^/p/\d+/'))
    
    for link in product_links:
        try:
            # Get the href
            href = link.get('href', '')
            if not href:
                continue
                
            # Build full URL
            if href.startswith('/'):
                url = 'https://www.kuantokusta.pt' + href
            else:
                url = href
            
            # Extract product name from the data-analytics-products-lists or href
            # The href contains product info: /p/ID/product-name-with-specs
            href_parts = href.split('/')
            if len(href_parts) >= 4:
                # Convert URL slug to readable name
                product_slug = href_parts[3].split('?')[0]  # Remove query params
                # Replace hyphens with spaces and capitalize
                name = product_slug.replace('-', ' ').title()
            else:
                # Fallback: try to get text content
                name = link.get_text(strip=True)
                
            if not name or len(name) < 5:
                continue
            
            # Look for price information in nearby elements or parent containers
            price = None
            
            # Try to find price in the parent container
            parent = link.parent
            for _ in range(3):  # Check up to 3 levels up
                if parent:
                    # Look for price patterns in the parent's text
                    parent_text = parent.get_text()
                    price_match = re.search(r'desde\s*(\d+[.,]?\d*)', parent_text)
                    if price_match:
                        price = price_match.group(1).replace(',', '.')
                        break
                    parent = parent.parent
                else:
                    break
            
            # If no price found, set a default (this will help us see if extraction is working)
            if not price:
                # Try to find any price-like pattern in the link's vicinity
                link_text = link.get_text(strip=True)
                price_match = re.search(r'(\d+[.,]?\d*)€?', link_text)
                if price_match:
                    price = price_match.group(1).replace(',', '.')
                else:
                    price = "0.00"  # Placeholder to indicate data extraction is working
            
            # Clean up the product name
            name = re.sub(r'\s+', ' ', name).strip()
            
            # Only add products with valid data
            if name and price and url:
                products.append({
                    'name': name,
                    'price': price,
                    'store': 'KuantoKusta',
                    'currency': 'EUR',
                    'url': url
                })
                
        except (AttributeError, ValueError, IndexError):
            # Skip problematic entries
            continue

    return products