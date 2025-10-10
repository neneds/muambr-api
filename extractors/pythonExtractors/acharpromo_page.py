import json
import time
import gzip
import sys
from bs4 import BeautifulSoup
import re

def extract_acharpromo_products(html_string):
    """
    Extract products from achar.promo search results
    The site appears to load content dynamically, so we need to handle both initial HTML and dynamic content
    """
    soup = BeautifulSoup(html_string, 'html.parser')
    products = []

    try:
        # First, try to extract from script tags that might contain JSON data
        script_tags = soup.find_all('script')
        for script in script_tags:
            if script.string:
                script_content = script.string.strip()
                
                # Look for JSON data containing product information
                if 'products' in script_content.lower() or 'items' in script_content.lower():
                    # Try to extract JSON from script content
                    json_matches = re.findall(r'\{.*?\}', script_content, re.DOTALL)
                    for match in json_matches:
                        try:
                            data = json.loads(match)
                            if isinstance(data, dict):
                                # Process potential product data
                                products.extend(extract_from_json_data(data))
                        except (json.JSONDecodeError, KeyError, TypeError):
                            continue

        # If no products found in JSON, try HTML parsing
        if not products:
            products = extract_from_html_structure(soup)

        # Remove duplicates based on URL
        seen_urls = set()
        unique_products = []
        for product in products:
            if product.get('url') not in seen_urls:
                seen_urls.add(product.get('url'))
                unique_products.append(product)

        return unique_products

    except Exception as e:
        return [{
            'error': f'Failed to parse achar.promo page: {str(e)}',
            'products': []
        }]

def extract_from_json_data(data):
    """Extract products from JSON data structure"""
    products = []
    
    try:
        # Handle different JSON structures that might contain product data
        if 'products' in data:
            for item in data['products']:
                product = parse_product_item(item)
                if product:
                    products.append(product)
        elif 'items' in data:
            for item in data['items']:
                product = parse_product_item(item)
                if product:
                    products.append(product)
        elif 'results' in data:
            for item in data['results']:
                product = parse_product_item(item)
                if product:
                    products.append(product)
    except (KeyError, TypeError):
        pass
    
    return products

def parse_product_item(item):
    """Parse individual product item from JSON data"""
    try:
        name = item.get('name') or item.get('title') or item.get('productName')
        price = item.get('price') or item.get('cost') or item.get('valor')
        url = item.get('url') or item.get('link') or item.get('productUrl')
        store = item.get('store') or item.get('shop') or item.get('merchant') or item.get('seller')
        
        # Clean price if it's a string
        if isinstance(price, str):
            # Remove currency symbols and extract numeric value
            price_clean = re.sub(r'[^\d,.]', '', price.replace(',', '.'))
            try:
                price = float(price_clean)
            except ValueError:
                price = None
        
        if name and price and url:
            return {
                'name': clean_product_name(name),
                'price': str(price),
                'store': store or 'AcharPromo',
                'currency': 'BRL',  # achar.promo is Brazilian
                'url': url if url.startswith('http') else f"https://achar.promo{url}"
            }
    except (KeyError, TypeError, AttributeError):
        pass
    
    return None

def extract_from_html_structure(soup):
    """Extract products from HTML structure when JSON parsing fails"""
    products = []
    
    try:
        # Common selectors for product listings on Brazilian e-commerce sites
        product_selectors = [
            'div[class*="product"]',
            'div[class*="item"]', 
            'article[class*="product"]',
            'div[class*="card"]',
            'li[class*="product"]',
            'div[data-testid*="product"]',
            'div[data-cy*="product"]'
        ]
        
        for selector in product_selectors:
            product_elements = soup.select(selector)
            if product_elements:
                for element in product_elements:
                    product = parse_html_product_element(element)
                    if product:
                        products.append(product)
                break  # Use first successful selector
        
        # If no products found with selectors, try finding by price patterns
        if not products:
            products = extract_by_price_pattern(soup)
            
    except Exception as e:
        pass
    
    return products

def parse_html_product_element(element):
    """Parse individual product element from HTML"""
    try:
        # Extract product name
        name_selectors = [
            'h3', 'h4', 'h5',
            '[class*="title"]', '[class*="name"]', '[class*="product"]',
            'a[title]', 'span[title]'
        ]
        
        name = None
        for selector in name_selectors:
            name_elem = element.select_one(selector)
            if name_elem:
                name = name_elem.get_text(strip=True) or name_elem.get('title')
                if name:
                    break
        
        # Extract price
        price_selectors = [
            '[class*="price"]', '[class*="valor"]', '[class*="cost"]',
            '[data-price]', 'span:contains("R$")', 'div:contains("R$")'
        ]
        
        price = None
        for selector in price_selectors:
            if ':contains(' in selector:
                # Handle text content search
                price_elems = element.find_all(text=re.compile(r'R\$'))
                for elem in price_elems:
                    price_match = re.search(r'R\$\s*([\d.,]+)', elem)
                    if price_match:
                        price = price_match.group(1).replace(',', '.')
                        break
            else:
                price_elem = element.select_one(selector)
                if price_elem:
                    price_text = price_elem.get_text(strip=True)
                    price_match = re.search(r'([\d.,]+)', price_text)
                    if price_match:
                        price = price_match.group(1).replace(',', '.')
                        break
        
        # Extract URL
        url = None
        link_elem = element.find('a', href=True)
        if link_elem:
            url = link_elem['href']
        
        # Extract store name
        store_selectors = [
            '[class*="store"]', '[class*="shop"]', '[class*="merchant"]',
            '[class*="seller"]', '[class*="vendor"]'
        ]
        
        store = None
        for selector in store_selectors:
            store_elem = element.select_one(selector)
            if store_elem:
                store = store_elem.get_text(strip=True)
                break
        
        if name and price:
            try:
                price_float = float(price)
                return {
                    'name': clean_product_name(name),
                    'price': str(price_float),
                    'store': store or 'AcharPromo',
                    'currency': 'BRL',
                    'url': url if url and url.startswith('http') else f"https://achar.promo{url or ''}"
                }
            except ValueError:
                pass
    
    except Exception:
        pass
    
    return None

def extract_by_price_pattern(soup):
    """Extract products by looking for price patterns in the entire page"""
    products = []
    
    try:
        # Find all text containing Brazilian currency
        price_pattern = re.compile(r'R\$\s*([\d.,]+)')
        all_text = soup.get_text()
        
        # This is a fallback method for dynamically loaded content
        # In a real implementation, we might need to use Selenium or similar
        prices = price_pattern.findall(all_text)
        
        if prices:
            # If we found prices but no structured data, return a placeholder
            return [{
                'name': 'Produto encontrado',
                'price': prices[0].replace(',', '.'),
                'store': 'AcharPromo',
                'currency': 'BRL',
                'url': 'https://achar.promo'
            }]
    
    except Exception:
        pass
    
    return products

def clean_product_name(name):
    """Clean and standardize product name"""
    if not name:
        return ""
    
    # Remove extra whitespace and normalize
    name = ' '.join(name.split())
    
    # Remove common unwanted patterns
    name = re.sub(r'\s*-\s*Frete\s+grátis.*', '', name, flags=re.IGNORECASE)
    name = re.sub(r'\s*\|\s*MercadoLivre.*', '', name, flags=re.IGNORECASE)
    
    return name.strip()

# Main execution
if __name__ == "__main__":
    import sys
    
    if len(sys.argv) != 2:
        print(json.dumps({"error": "HTML file path required"}))
        sys.exit(1)
    
    html_file_path = sys.argv[1]
    
    try:
        # Try to read as binary first to check for gzip compression
        with open(html_file_path, 'rb') as file:
            raw_content = file.read()
        
        # Check if content is gzip compressed
        if raw_content.startswith(b'\x1f\x8b'):
            # Decompress gzip content
            html_content = gzip.decompress(raw_content).decode('utf-8')
        else:
            # Try to decode as UTF-8 directly
            try:
                html_content = raw_content.decode('utf-8')
            except UnicodeDecodeError:
                # Try other common encodings
                for encoding in ['latin-1', 'iso-8859-1', 'cp1252']:
                    try:
                        html_content = raw_content.decode(encoding)
                        break
                    except UnicodeDecodeError:
                        continue
                else:
                    raise UnicodeDecodeError("Unable to decode HTML content with any common encoding")
        
        products = extract_acharpromo_products(html_content)
        
        # Output results as JSON
        print(json.dumps({
            "products": products,
            "total": len(products),
            "source": "achar.promo"
        }, ensure_ascii=False, indent=2))
    
    except FileNotFoundError:
        print(json.dumps({"error": f"File not found: {html_file_path}"}))
    except Exception as e:
        print(json.dumps({"error": f"Failed to process file: {str(e)}"}))