#!/usr/bin/env python3

import sys
import os
import json

# Add the extractors directory to the path
sys.path.append(os.path.join(os.path.dirname(__file__), 'extractors', 'pythonExtractors'))

try:
    from kuantokusta_page import extract_kuantokusta_products
    print("✓ Successfully imported kuantokusta_page module")
except ImportError as e:
    print(f"✗ Failed to import kuantokusta_page: {e}")
    sys.exit(1)

try:
    from mercadolivre_page import extract_mercadolivre_products
    print("✓ Successfully imported mercadolivre_page module")
except ImportError as e:
    print(f"✗ Failed to import mercadolivre_page: {e}")

# Test with sample HTML
sample_html = """
<html>
<body>
<a href="/p/12345/iphone-15-128gb-blue">iPhone 15</a>
<div class="price">desde 899€</div>
</body>
</html>
"""

print("\nTesting KuantoKusta extraction with sample HTML...")
try:
    products = extract_kuantokusta_products(sample_html)
    print(f"Found {len(products)} products:")
    for product in products:
        print(f"  - {product}")
except Exception as e:
    print(f"✗ KuantoKusta extraction failed: {e}")

print("\nPython environment check complete.")