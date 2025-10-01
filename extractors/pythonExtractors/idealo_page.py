import json
from bs4 import BeautifulSoup

def extract_idealo_products(html_string):
    soup = BeautifulSoup(html_string, 'html.parser')
    products = []

    # Find all product offer rows in the Comparador de precios section
    for row in soup.find_all('li', class_='productOffers-listItem'):
        # Product name
        name_tag = row.find('span', class_='productOffers-listItemTitleInner')
        name = name_tag.get_text(strip=True) if name_tag else None

        # Store name
        store_tag = row.find('span', class_='productOffers-listItemShop')
        store = store_tag.get_text(strip=True) if store_tag else "Unknown Store"

        # Price and currency
        price_tag = row.find('a', class_='productOffers-listItemOfferPrice')
        if price_tag:
            price_text = price_tag.get_text(strip=True)
            import re
            match = re.search(r'([\d.,]+)\s*([A-Z€$£€]+)', price_text)
            if match:
                price = match.group(1)  # Keep as string
                currency = match.group(2)
            else:
                price = price_text
                currency = "EUR"  # Default for Spain
        else:
            price = None
            currency = "EUR"

        # Product URL (prefer the price link, else fallback to title link)
        url_tag = price_tag if price_tag and price_tag.has_attr('href') else row.find('a', class_='productOffers-listItemTitle')
        url = url_tag['href'] if url_tag and url_tag.has_attr('href') else None
        if url and not url.startswith('https://'):
            url = f'https://www.idealo.es{url}'

        # Only add products with valid data
        if name and price and url:
            products.append({
                'name': name,
                'price': price,
                'store': store,
                'currency': currency,
                'url': url
            })

    return products