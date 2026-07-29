const API_URL = window.API_URL || 'http://localhost:8080';

function formatPrice(price, currency) {
  return `${parseFloat(price).toFixed(2)} ${currency}`;
}

function renderShopHeader(shop) {
  const container = document.getElementById('shopProfileHeader');
  document.title = `${shop.name} — FlipTheTable`;

  const banner = shop.banner_url || 'https://picsum.photos/seed/shop-banner/1200/240';
  const avatar = shop.avatar_url || 'https://picsum.photos/seed/shop-avatar/160/160';

  container.innerHTML = `
    <div class="shop-profile-banner" style="background-image:url('${banner}')"></div>
    <div class="shop-profile-summary">
      <img class="shop-profile-avatar" src="${avatar}" alt="${escapeHtml(shop.name)}" />
      <div>
        <h1>${escapeHtml(shop.name)}</h1>
        <p class="shop-profile-meta">${shop.listings_count} ${shop.listings_count === 1 ? 'oferta' : 'ofert'} · ${shop.sales_count} sprzedanych</p>
        ${shop.description ? `<p class="shop-profile-desc">${escapeHtml(shop.description)}</p>` : ''}
      </div>
    </div>
  `;
}

function renderShopListings(items) {
  const grid = document.getElementById('listingGrid');
  const emptyState = document.getElementById('emptyState');
  const resultCount = document.getElementById('resultCount');

  grid.innerHTML = '';

  if (items.length === 0) {
    emptyState.style.display = 'block';
    resultCount.textContent = '';
    return;
  }

  emptyState.style.display = 'none';
  resultCount.textContent = `${items.length} ${items.length === 1 ? 'oferta' : 'ofert'}`;

  items.forEach((item) => {
    const card = document.createElement('a');
    card.className = 'listing-card';
    card.href = `listing.html?id=${item.id}`;

    const img = item.primary_photo || 'https://picsum.photos/seed/placeholder/600/600';
    const outOfStock = item.quantity === 0;

    card.innerHTML = `
      <img src="${img}" alt="${escapeHtml(item.title)}" loading="lazy" />
      <div class="listing-card-body">
        <p class="listing-card-title">${escapeHtml(item.title)}</p>
        <p class="listing-card-price">${formatPrice(item.price, item.currency)}</p>
        ${outOfStock ? '<p class="listing-card-stock">Brak w magazynie</p>' : ''}
      </div>
    `;

    grid.appendChild(card);
  });
}

async function loadShopProfile() {
  const id = new URLSearchParams(window.location.search).get('id');
  if (!id) {
    document.getElementById('shopProfileHeader').innerHTML = '<p style="padding:24px;">Brak identyfikatora sklepu.</p>';
    return;
  }

  try {
    const shopRes = await fetch(`${API_URL}/shops/${id}`);
    if (!shopRes.ok) {
      document.getElementById('shopProfileHeader').innerHTML = '<p style="padding:24px;">Sklep nie znaleziony.</p>';
      return;
    }
    const shop = await shopRes.json();
    renderShopHeader(shop);

    const listingsRes = await fetch(`${API_URL}/listings?shop_id=${id}&page_size=100`);
    const data = await listingsRes.json();
    renderShopListings(data.items || []);
  } catch (err) {
    document.getElementById('shopProfileHeader').innerHTML = '<p style="padding:24px;">Błąd ładowania sklepu.</p>';
  }
}

updateHeaderForAuth();
loadShopProfile();
