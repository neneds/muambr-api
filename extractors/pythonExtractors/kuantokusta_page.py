import json
from bs4 import BeautifulSoup

def extract_kuantokusta_products(html_string):
    soup = BeautifulSoup(html_string, 'html.parser')
    products = []

    # Find all offer cards by data-test-id
    offer_cards = soup.find_all('div', attrs={'data-test-id': 'offer-card'})
    for card in offer_cards:
        # Product name
        name_tag = card.find('h3', attrs={'data-test-id': 'offer-product-name'})
        name = name_tag.text.strip() if name_tag else None

        # Store name
        shop = None
        img = card.find('img')
        if img and img.has_attr('alt'):
            shop = img['alt']
        else:
            shop = card.get('data-store-name', 'Unknown Store')

        # Price
        price_tag = card.find('span', attrs={'data-test-id': 'offer-total-price'})
        if not price_tag:
            price_tag = card.find('span', attrs={'data-test-id': 'offer-price'})
        
        price = None
        if price_tag:
            price_text = price_tag.text.replace('€', '').replace(',', '.').strip()
            # Keep price as string
            price = price_text

        currency = 'EUR'  # KuantoKusta is Portugal-specific

        # URL
        url = None
        # Try to find a <a> tag wrapping the offer card
        a_tag = card.find_parent('a', href=True)
        if a_tag:
            url = a_tag['href']
            if url.startswith('/'):
                url = 'https://www.kuantokusta.pt' + url
        # Fallback: try to find any <a> inside the card with href
        if not url:
            a_tag_inner = card.find('a', href=True)
            if a_tag_inner:
                url = a_tag_inner['href']
                if url.startswith('/'):
                    url = 'https://www.kuantokusta.pt' + url

        # Only add products with valid data
        if name and price and url:
            products.append({
                'name': name,
                'price': price,
                'store': shop,
                'currency': currency,
                'url': url
            })

    return products