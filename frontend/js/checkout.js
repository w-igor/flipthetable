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
  const listingIds = Object.keys(cart);

  if (listingIds.length === 0) {
    document.getElementById('checkoutEmpty').style.display = 'block';
    document.getElementById('checkoutLayout').style.display = 'none';
    return null;
  }

  const results = await Promise.all(
    listingIds.map((id) =>
      fetch(`${API_URL}/listings/${id}`)
        .then((res) => (res.ok ? res.json() : null))
        .catch(() => null)
    )
  );

  const shopGroups = {};
  let total = 0;

  results.forEach((listing, idx) => {
    if (!listing) return;
    const listingId = listingIds[idx];
    const qty = cart[listingId];
    const lineTotal = parseFloat(listing.price) * qty;
    total += lineTotal;

    if (!shopGroups[listing.shop_id]) {
      shopGroups[listing.shop_id] = { shopName: listing.shop_name, lines: [] };
    }
    shopGroups[listing.shop_id].lines.push({ listing, qty, lineTotal });
  });

  const summaryEl = document.getElementById('summaryContent');
  summaryEl.innerHTML = '';

  Object.values(shopGroups).forEach((group) => {
    const groupEl = document.createElement('div');
    groupEl.className = 'summary-shop-group';
    const linesHtml = group.lines
      .map(({ listing, qty, lineTotal }) => {
        const img = listing.primary_photo || 'https://picsum.photos/seed/placeholder/200/200';
        return `
          <div class="summary-line">
            <img src="${img}" alt="${escapeHtml(listing.title)}" />
            <span class="summary-line-title">${escapeHtml(listing.title)} × ${qty}</span>
            <span>${lineTotal.toFixed(2)} PLN</span>
          </div>
        `;
      })
      .join('');

    groupEl.innerHTML = `<p class="summary-shop-name">${escapeHtml(group.shopName)}</p>${linesHtml}`;
    summaryEl.appendChild(groupEl);
  });

  document.getElementById('summaryTotal').textContent = `${total.toFixed(2)} PLN`;

  return { cart, listingIds };
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
    const items = Object.entries(cart).map(([listing_id, quantity]) => ({ listing_id, quantity }));

    if (items.length === 0) {
      showError('Koszyk jest pusty.');
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
    submitBtn.textContent = 'Przetwarzanie...';

    try {
      const res = await authFetch('/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ items, shipping_addr, note }),
      });
      const data = await res.json();

      if (!res.ok) {
        showError(data.message || 'Nie udało się złożyć zamówienia');
        submitBtn.disabled = false;
        submitBtn.textContent = 'Złóż zamówienie';
        return;
      }

      localStorage.removeItem('cart');
      const orderIds = data.orders.map((o) => o.id).join(',');
      window.location.href = `order-confirmation.html?ids=${orderIds}`;
    } catch (err) {
      showError('Nie udało się połączyć z serwerem.');
      submitBtn.disabled = false;
      submitBtn.textContent = 'Złóż zamówienie';
    }
  });
}

initCheckout();
