import json
from bs4 import BeautifulSoup
import re

def extract_kelkoo_products(html_string):
    soup = BeautifulSoup(html_string, 'html.parser')
    products = []
    
    # Check if the page is blocked by anti-bot protection
    if "Please enable JS" in html_string or "captcha" in html_string.lower():
        return products

    # Kelkoo uses JSON-LD structured data for products
    # Find all script tags with type="application/ld+json"
    json_ld_scripts = soup.find_all('script', type='application/ld+json')
    
    for script in json_ld_scripts:
        try:
            if not script.string:
                continue
                
            data = json.loads(script.string)
            
            # Handle both single objects and arrays
            if isinstance(data, list):
                items = data
            elif isinstance(data, dict):
                if '@graph' in data:
                    items = data['@graph']
                elif '@type' in data:
                    items = [data]
                else:
                    continue
            else:
                continue
            
            # Extract products from JSON-LD data
            for item in items:
                try:
                    if item.get('@type') == 'Product':
                        name = item.get('name', '').strip()
                        if not name or len(name) < 5:
                            continue
                            
                        # Extract offer information
                        offers = item.get('offers', {})
                        if isinstance(offers, list):
                            # Take the first offer if it's a list
                            offers = offers[0] if offers else {}
                            
                        price = offers.get('price', '')
                        currency = offers.get('priceCurrency', 'EUR')
                        
                        # Convert price to string format
                        if isinstance(price, (int, float)):
                            price = str(float(price))
                        elif isinstance(price, str):
                            # Extract numeric price handling European formatting
                            price_match = re.search(r'(\d{1,3}(?:[.,]\d{3})*(?:[.,]\d{2})?)', str(price))
                            if price_match:
                                price_str = price_match.group(1)
                                # Handle European formatting: 1.049,99 -> 1049.99
                                if ',' in price_str and '.' in price_str:
                                    # European format with thousands separator
                                    price = price_str.replace('.', '').replace(',', '.')
                                elif ',' in price_str:
                                    # Only comma (could be decimal or thousands)
                                    parts = price_str.split(',')
                                    if len(parts) == 2 and len(parts[1]) == 2:
                                        # Decimal comma: 1049,99 -> 1049.99
                                        price = price_str.replace(',', '.')
                                    else:
                                        # Thousands comma: 1,049 -> 1049
                                        price = price_str.replace(',', '')
                                else:
                                    # Only dots or plain number
                                    if '.' in price_str and len(price_str.split('.')[-1]) == 2:
                                        # Decimal dot: 1049.99
                                        price = price_str
                                    else:
                                        # Thousands separator: 1.049 -> 1049
                                        price = price_str.replace('.', '')
                            else:
                                continue
                        else:
                            continue
                            
                        # Use consistent store name for Kelkoo
                        store_name = 'Kelkoo'
                            
                        # Get product URL
                        product_url = item.get('url', offers.get('url', 'https://www.kelkoo.es'))
                        
                        # Clean up product name (remove excessive details)
                        name = re.sub(r'\s+', ' ', name).strip()
                        if len(name) > 80:
                            name = name[:77] + "..."
                            
                        products.append({
                            'name': name,
                            'price': price,
                            'store': store_name,
                            'currency': currency,
                            'url': product_url
                        })
                        
                except (KeyError, TypeError, ValueError):
                    continue
                    
        except (json.JSONDecodeError, TypeError):
            continue

    return products# Test function for debugging
if __name__ == "__main__":
    # Test with sample HTML
    test_html = """
    <div class="product-item">
        <h3>Sony WH-1000XM6</h3>
        <div class="price">299.99 €</div>
        <div class="shop">MediaMarkt</div>
    </div>
    """
    result = extract_kelkoo_products(test_html)
    print(json.dumps(result, indent=2))