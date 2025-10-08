import json
from bs4 import BeautifulSoup
import re

def extract_mercadolivre_products(html_string):
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
                                'store': 'Mercado Livre',
                                'currency': offers.get('priceCurrency', 'BRL'),
                                'url': url
                            })
        except (json.JSONDecodeError, KeyError, TypeError):
            continue

    # If JSON-LD didn't work, fallback to HTML parsing
    if not products:
        # Look for product cards in the listing
        product_selectors = [
            'div.ui-search-result',
            'li.ui-search-layout__item',
            'div.poly-card',
            'article[data-testid="result"]',
            'div[class*="ui-search-result"]'
        ]
        
        product_cards = []
        for selector in product_selectors:
            product_cards = soup.select(selector)
            if product_cards:
                break

        for card in product_cards:
            try:
                # Extract product name
                name = None
                name_selectors = [
                    'h2.ui-search-item__title',
                    'h2.poly-component__title',
                    'a.ui-search-link',
                    'h2[class*="title"]',
                    'a[class*="item__title"]'
                ]
                
                for name_sel in name_selectors:
                    name_element = card.select_one(name_sel)
                    if name_element:
                        name = name_element.get_text(strip=True)
                        if name and len(name) > 5:
                            break
                
                # Extract price
                price = None
                currency = 'BRL'
                
                price_selectors = [
                    'span.andes-money-amount__fraction',
                    'span.price-tag-fraction',
                    'div.ui-search-price__second-line span.andes-money-amount__fraction',
                    'span[class*="price-tag"]',
                    'span[class*="money-amount"]'
                ]
                
                for price_sel in price_selectors:
                    price_element = card.select_one(price_sel)
                    if price_element:
                        price_text = price_element.get_text(strip=True)
                        # Extract price handling Brazilian format (R$ 3.997,99)
                        price_match = re.search(r'([\d.,]+)', price_text)
                        if price_match:
                            price_str = price_match.group(0)
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
                            
                            # Check for currency symbol
                            currency_element = card.select_one('span.andes-money-amount__currency-symbol')
                            if currency_element:
                                currency_text = currency_element.get_text(strip=True)
                                if currency_text == 'R$':
                                    currency = 'BRL'
                            break
                
                # Extract URL
                url = 'https://www.mercadolivre.com.br'
                link_element = card.select_one('a[href]')
                if link_element:
                    href = link_element.get('href', '')
                    if href.startswith('http'):
                        url = href
                    elif href.startswith('/'):
                        url = 'https://www.mercadolivre.com.br' + href
                    else:
                        url = 'https://www.mercadolivre.com.br/' + href
                
                # Only add products with valid name and price
                if name and price:
                    products.append({
                        'name': name,
                        'price': price,
                        'store': 'Mercado Livre',
                        'currency': currency,
                        'url': url
                    })
                    
            except (AttributeError, ValueError, KeyError):
                # Skip problematic cards
                continue

    return products

# Test function for debugging
if __name__ == '__main__':
    # Test with sample HTML
    test_html = '''
    <div class="ui-search-result">
        <h2 class="ui-search-item__title">Sony WH-1000XM6</h2>
        <div class="ui-search-price__second-line">
            <span class="andes-money-amount__currency-symbol">R$</span>
            <span class="andes-money-amount__fraction">3.997</span>
        </div>
        <a href="/produto.mercadolivre.com.br/MLB-123456">Ver produto</a>
    </div>
    '''
    result = extract_mercadolivre_products(test_html)
    print(json.dumps(result, indent=2, ensure_ascii=False))
