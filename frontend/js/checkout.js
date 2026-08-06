// Checkout Process Management
// Handles order creation, payment processing, and order confirmation

// Safely escapes HTML to prevent XSS attacks
function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// Displays an error banner message
function showError(message) {
  const banner = document.getElementById('errorBanner');
  banner.textContent = message;
  banner.classList.add('show');
}

// Loads and displays the checkout order summary
// Fetches listing details, calculates totals, and groups items by shop
async function loadSummary() {
  const cart = getCart();
  const entries = Object.entries(cart);

  // Show empty cart message if no items
  if (entries.length === 0) {
    document.getElementById('checkoutEmpty').style.display = 'block';
    document.getElementById('checkoutLayout').style.display = 'none';
    return null;
  }

  // Batch fetch all listing details
  const uniqueListingIds = [...new Set(entries.map(([, e]) => e.listingId))];
  const listingResults = await Promise.all(
    uniqueListingIds.map((id) =>
      fetch(`${API_URL}/listings/${id}`)
        .then((res) => (res.ok ? res.json() : null))
        .catch(() => null)
    )
  );
  const listingsById = Object.fromEntries(uniqueListingIds.map((id, i) => [id, listingResults[i]]));

  // Group items by shop for separate order creation
  const shopGroups = {};
  let total = 0;

  entries.forEach(([, entry]) => {
    const listing = listingsById[entry.listingId];
    if (!listing) return;
    const line = resolveCartLine(listing, entry);
    if (!line) return;
    const lineTotal = line.price * entry.quantity;
    total += lineTotal;

    if (!shopGroups[listing.shop_id]) {
      shopGroups[listing.shop_id] = { shopName: listing.shop_name, lines: [], shippingPrice: 0 };
    }
    if (listing.shipping) {
      shopGroups[listing.shop_id].shippingPrice = Math.max(shopGroups[listing.shop_id].shippingPrice, parseFloat(listing.shipping.price));
    }
    shopGroups[listing.shop_id].lines.push({ listing, entry, line, lineTotal });
  });

  const summaryEl = document.getElementById('summaryContent');
  summaryEl.innerHTML = '';

  Object.values(shopGroups).forEach((group) => {
    total += group.shippingPrice;

    const groupEl = document.createElement('div');
    groupEl.className = 'summary-shop-group';
    const linesHtml = group.lines
      .map(({ listing, entry, line, lineTotal }) => {
        const img = listing.primary_photo || 'https://picsum.photos/seed/placeholder/200/200';
        const variantSuffix = line.label ? ` (${escapeHtml(line.label)})` : '';
        return `
          <div class="summary-line">
            <img src="${img}" alt="${escapeHtml(listing.title)}" />
            <span class="summary-line-title">${escapeHtml(listing.title)}${variantSuffix} × ${entry.quantity}</span>
            <span>${lineTotal.toFixed(2)} PLN</span>
          </div>
        `;
      })
      .join('');
    const shippingHtml = `<div class="summary-line summary-shipping-line"><span class="summary-line-title">${t('checkout.shipping_line')}</span><span>${group.shippingPrice.toFixed(2)} PLN</span></div>`;

    groupEl.innerHTML = `<p class="summary-shop-name">${escapeHtml(group.shopName)}</p>${linesHtml}${shippingHtml}`;
    summaryEl.appendChild(groupEl);
  });

  document.getElementById('summaryTotal').textContent = `${total.toFixed(2)} PLN`;

  return { cart };
}

// Initializes the checkout page with authentication check and order processing
function initCheckout() {
  const token = getAccessToken();
  if (!token) {
    window.location.href = `login.html?redirect=checkout.html`;
    return;
  }

  loadSummary();

  document.getElementById('checkoutForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    document.getElementById('errorBanner').classList.remove('show');

    const cart = getCart();
    const items = Object.values(cart).map((entry) => ({
      listing_id: entry.listingId,
      variant_sku_id: entry.variantSkuId || undefined,
      quantity: entry.quantity,
    }));

    if (items.length === 0) {
      showError(t('checkout.empty_cart'));
      return;
    }

    const shipping_addr = {
      full_name: document.getElementById('fullName').value.trim(),
      address: document.getElementById('address').value.trim(),
      city: document.getElementById('city').value.trim(),
      postal_code: document.getElementById('postalCode').value.trim(),
      country: document.getElementById('country').value.trim(),
      phone: document.getElementById('phone').value.trim(),
    };
    const note = document.getElementById('note').value.trim();

    const submitBtn = document.getElementById('submitBtn');
    submitBtn.disabled = true;
    submitBtn.textContent = t('checkout.processing');

    try {
      const res = await authFetch('/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ items, shipping_addr, note }),
      });
      const data = await res.json();

      if (!res.ok) {
        showError(data.message || t('checkout.error_order_failed'));
        submitBtn.disabled = false;
        submitBtn.textContent = t('checkout.submit');
        return;
      }

      localStorage.removeItem('cart');
      const orderIds = data.orders.map((o) => o.id);

      submitBtn.textContent = t('checkout.processing_payment');
      const sessionRes = await authFetch('/orders/checkout-session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ order_ids: orderIds }),
      });
      const sessionData = await sessionRes.json();

      if (!sessionRes.ok || !sessionData.url) {
        window.location.href = `order-confirmation.html?ids=${orderIds.join(',')}&paymentIssue=1`;
        return;
      }

      window.location.href = sessionData.url;
    } catch (err) {
      showError(t('common.error_connect'));
      submitBtn.disabled = false;
      submitBtn.textContent = t('checkout.submit');
    }
  });
}

initCheckout();
