// Individual Seller Shop Profile Page
// Displays seller information, shop banner/avatar, and their product listings

// Formats a price value with currency code
function formatPrice(price, currency) {
  return `${parseFloat(price).toFixed(2)} ${currency}`;
}

// Renders the shop header with banner, avatar, and seller contact button
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
        <p class="shop-profile-meta">${tn('shop_profile.listings_count', shop.listings_count)} · ${shop.sales_count} ${t('shop_profile.sales_suffix')}</p>
        ${shop.description ? `<p class="shop-profile-desc">${escapeHtml(shop.description)}</p>` : ''}
        ${contactSellerButtonHtml(shop)}
      </div>
    </div>
  `;
}

function contactSellerButtonHtml(shop) {
  const user = getCurrentUser();
  if (!user || user.id === shop.owner_id) return '';
  const url = `messages.html?with=${shop.owner_id}&name=${encodeURIComponent(shop.name)}`;
  return `<a href="${url}" class="shop-profile-contact-btn">${t('shop_profile.contact_seller')}</a>`;
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
  resultCount.textContent = tn('shop_profile.listings_count', items.length);

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
        ${outOfStock ? `<p class="listing-card-stock">${t('shop.out_of_stock')}</p>` : ''}
      </div>
    `;

    grid.appendChild(card);
  });
}

async function loadShopProfile() {
  const id = new URLSearchParams(window.location.search).get('id');
  if (!id) {
    document.getElementById('shopProfileHeader').innerHTML = `<p style="padding:24px;">${t('shop_profile.no_shop_id')}</p>`;
    return;
  }

  try {
    const shopRes = await fetch(`${API_URL}/shops/${id}`);
    if (!shopRes.ok) {
      document.getElementById('shopProfileHeader').innerHTML = `<p style="padding:24px;">${t('shop_profile.not_found')}</p>`;
      return;
    }
    const shop = await shopRes.json();
    renderShopHeader(shop);

    const listingsRes = await fetch(`${API_URL}/listings?shop_id=${id}&page_size=100`);
    const data = await listingsRes.json();
    renderShopListings(data.items || []);
  } catch (err) {
    document.getElementById('shopProfileHeader').innerHTML = `<p style="padding:24px;">${t('shop_profile.error_loading')}</p>`;
  }
}

function onPageLocaleChange() {
  loadShopProfile();
}

updateHeaderForAuth();
loadShopProfile();
