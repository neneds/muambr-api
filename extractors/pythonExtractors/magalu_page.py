import json
from bs4 import BeautifulSoup
import re

def extract_magalu_products(html_string):
    soup = BeautifulSoup(html_string, 'html.parser')
    products = []

    # Try to extract from JSON-LD structured data first (more reliable)
    json_ld_scripts = soup.find_all('script', type='application/ld+json')
    for script in json_ld_scripts:
        try:
            data = json.loads(script.string)
            if isinstance(data, dict) and '@graph' in data:
                for item in data['@graph']:
                    if item.get('@type') == 'Product':
                        name = item.get('name')
                        offers = item.get('offers', {})
                        price = offers.get('price')
                        url = offers.get('url')
                        
                        # Only add valid products
                        if name and price and url:
                            products.append({
                                'name': name,
                                'price': str(price),
                                'store': 'Magazine Luiza',
                                'currency': offers.get('priceCurrency', 'BRL'),
                                'url': url
                            })
        except (json.JSONDecodeError, KeyError, TypeError):
            continue

    # If JSON-LD didn't work, fallback to HTML parsing
    if not products:
        # Look for product cards - Magazine Luiza uses various selectors
        product_selectors = [
            '[data-testid*="product"]',
            'li[data-testid*="product-card"]',
            'div[data-testid*="product-card"]',
            'article[data-testid*="product"]',
            'div.sc-*[data-testid]',  # Styled components pattern
            'li.sc-*[data-testid]',
            '[class*="ProductCard"]',
            '[class*="product-card"]'
        ]
        
        product_cards = []
        for selector in product_selectors:
            try:
                product_cards = soup.select(selector)
                if product_cards:
                    break
            except:
                continue

        # Also try to look for any elements with testid containing product
        if not product_cards:
            all_testid_elements = soup.find_all(attrs={'data-testid': re.compile(r'product', re.I)})
            product_cards = all_testid_elements

        for card in product_cards:
            try:
                # Extract product name - Magazine Luiza commonly uses these patterns
                name = None
                name_selectors = [
                    '[data-testid*="product-title"]',
                    '[data-testid*="product-name"]',
                    'h2[data-testid]',
                    'h3[data-testid]',
                    'a[title]',
                    'img[alt]',
                    '[class*="title"]',
                    '[class*="name"]'
                ]
                
                for name_sel in name_selectors:
                    try:
                        name_elements = card.select(name_sel)
                        for element in name_elements:
                            if element.get('title'):
                                name = element.get('title').strip()
                            elif element.get('alt'):
                                name = element.get('alt').strip()
                            elif element.get_text(strip=True):
                                name = element.get_text(strip=True)
                            
                            if name and len(name) > 5 and 'iphone' in name.lower():
                                break
                        if name and len(name) > 5:
                            break
                    except:
                        continue
                
                # Extract price - Magazine Luiza price patterns
                price = None
                currency = 'BRL'
                
                price_selectors = [
                    '[data-testid*="price"]',
                    '[data-testid*="value"]',
                    'span[class*="price"]',
                    'div[class*="price"]',
                    'p[class*="price"]',
                    '[class*="Price"]',
                    '[class*="Value"]'
                ]
                
                for price_sel in price_selectors:
                    try:
                        price_elements = card.select(price_sel)
                        for price_element in price_elements:
                            price_text = price_element.get_text(strip=True)
                            if price_text and ('R$' in price_text or re.search(r'\d', price_text)):
                                # Handle Brazilian price format: R$ 3.997,99
                                price_match = re.search(r'R?\$?\s?(\d{1,3}(?:\.\d{3})*(?:,\d{2})?)', price_text)
                                if price_match:
                                    price_str = price_match.group(1)
                                    # Handle Brazilian formatting: 3.997,99 -> 3997.99
                                    if ',' in price_str and '.' in price_str:
                                        # Brazilian format with thousands separator: 3.997,99 -> 3997.99
                                        price = price_str.replace('.', '').replace(',', '.')
                                    elif ',' in price_str:
                                        # Only comma - treat as decimal separator: 3997,99 -> 3997.99
                                        price = price_str.replace(',', '.')
                                    elif '.' in price_str:
                                        # Check if it's thousands separator or decimal
                                        parts = price_str.split('.')
                                        if len(parts) == 2 and len(parts[1]) == 2:
                                            # Decimal point: 3997.99
                                            price = price_str
                                        else:
                                            # Thousands separator: 3.997 -> 3997
                                            price = price_str.replace('.', '')
                                    else:
                                        # Plain number: 3997
                                        price = price_str
                                    break
                        if price:
                            break
                    except:
                        continue
                
                # Extract URL
                url = 'https://www.magazineluiza.com.br'
                link_selectors = [
                    'a[href]',
                    'a[data-testid]'
                ]
                
                for link_sel in link_selectors:
                    try:
                        link_element = card.select_one(link_sel)
                        if link_element:
                            href = link_element.get('href', '')
                            if href:
                                if href.startswith('http'):
                                    url = href
                                elif href.startswith('/'):
                                    url = 'https://www.magazineluiza.com.br' + href
                                else:
                                    url = 'https://www.magazineluiza.com.br/' + href
                                break
                    except:
                        continue
                
                # Only add products with valid name and price
                if name and price and float(price) > 0:
                    products.append({
                        'name': name,
                        'price': price,
                        'store': 'Magazine Luiza',
                        'currency': currency,
                        'url': url
                    })
                    
            except (AttributeError, ValueError, KeyError):
                # Skip problematic cards
                continue

    return products

# Test function for debugging
if __name__ == '__main__':
    # Test with sample HTML structure
    test_html = '''
    <div data-testid="product-card">
        <h3 data-testid="product-title">iPhone 16 Pro Max 256GB</h3>
        <div data-testid="product-price">
            <span>R$ 7.999,99</span>
        </div>
        <a href="/p/iphone-16-pro-max/12345">Ver produto</a>
    </div>
    '''
    result = extract_magalu_products(test_html)
    print(json.dumps(result, indent=2, ensure_ascii=False))