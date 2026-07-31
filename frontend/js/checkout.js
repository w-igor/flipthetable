function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

function showError(message) {
  const banner = document.getElementById('errorBanner');
  banner.textContent = message;
  banner.classList.add('show');
}

async function loadSummary() {
  const cart = JSON.parse(localStorage.getItem('cart') || '{}');
  const entries = Object.entries(cart);

  if (entries.length === 0) {
    document.getElementById('checkoutEmpty').style.display = 'block';
    document.getElementById('checkoutLayout').style.display = 'none';
    return null;
  }

  const uniqueListingIds = [...new Set(entries.map(([, e]) => e.listingId))];
  const listingResults = await Promise.all(
    uniqueListingIds.map((id) =>
      fetch(`${API_URL}/listings/${id}`)
        .then((res) => (res.ok ? res.json() : null))
        .catch(() => null)
    )
  );
  const listingsById = Object.fromEntries(uniqueListingIds.map((id, i) => [id, listingResults[i]]));

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

    const cart = JSON.parse(localStorage.getItem('cart') || '{}');
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

    const cardholderName = document.getElementById('cardholderName').value.trim();
    const cardNumber = document.getElementById('cardNumber').value.trim();
    const cardCvc = document.getElementById('cardCvc').value.trim();
    const expiryMatch = document.getElementById('cardExpiry').value.trim().match(/^(\d{1,2})\s*\/\s*(\d{2,4})$/);
    if (!expiryMatch) {
      showError(t('checkout.invalid_expiry'));
      return;
    }
    const expMonth = parseInt(expiryMatch[1], 10);
    const expYear = expiryMatch[2].length === 2 ? 2000 + parseInt(expiryMatch[2], 10) : parseInt(expiryMatch[2], 10);

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
      const paymentResults = await Promise.all(
        orderIds.map((id) =>
          authFetch(`/orders/${id}/pay`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              cardholder_name: cardholderName,
              card_number: cardNumber,
              exp_month: expMonth,
              exp_year: expYear,
              cvc: cardCvc,
            }),
          })
            .then((r) => r.json())
            .catch(() => ({ payment_status: 'failed' }))
        )
      );

      const anyFailed = paymentResults.some((p) => p.payment_status !== 'completed');
      const suffix = anyFailed ? '&paymentIssue=1' : '';
      window.location.href = `order-confirmation.html?ids=${orderIds.join(',')}${suffix}`;
    } catch (err) {
      showError(t('common.error_connect'));
      submitBtn.disabled = false;
      submitBtn.textContent = t('checkout.submit');
    }
  });
}

initCheckout();
