function requireLogin() {
  const user = getCurrentUser();
  if (!user) {
    window.location.href = 'login.html?redirect=favorites.html';
    return null;
  }
  return user;
}

function formatPrice(price, currency) {
  return `${parseFloat(price).toFixed(2)} ${currency}`;
}

function renderFavorites(items) {
  const grid = document.getElementById('favoritesGrid');
  const emptyState = document.getElementById('favoritesEmpty');

  if (items.length === 0) {
    grid.innerHTML = '';
    emptyState.style.display = 'block';
    return;
  }
  emptyState.style.display = 'none';

  grid.innerHTML = items
    .map((item) => {
      const img = item.primary_photo || 'https://picsum.photos/seed/placeholder/600/600';
      return `
        <a class="listing-card" href="listing.html?id=${item.id}">
          <button class="listing-card-fav-btn active" type="button" data-id="${item.id}" title="${t('favorites.remove_title')}">♥</button>
          <img src="${img}" alt="${escapeHtml(item.title)}" loading="lazy" />
          <div class="listing-card-body">
            <p class="listing-card-shop">${escapeHtml(item.shop_name)}</p>
            <p class="listing-card-title">${escapeHtml(item.title)}</p>
            <p class="listing-card-price">${formatPrice(item.price, item.currency)}</p>
          </div>
        </a>
      `;
    })
    .join('');

  grid.querySelectorAll('.listing-card-fav-btn').forEach((btn) => {
    btn.addEventListener('click', async (e) => {
      e.preventDefault();
      e.stopPropagation();
      btn.disabled = true;
      try {
        await authFetch(`/favorites/${btn.dataset.id}`, { method: 'DELETE' });
        loadFavorites();
      } catch (err) {
        btn.disabled = false;
      }
    });
  });
}

async function loadFavorites() {
  const grid = document.getElementById('favoritesGrid');
  grid.innerHTML = `<p class="orders-empty">${t('common.loading')}</p>`;
  try {
    const res = await authFetch('/favorites');
    const items = await res.json();
    if (!res.ok) {
      grid.innerHTML = `<p class="orders-empty">${t('favorites.err_fetch')}</p>`;
      return;
    }
    renderFavorites(items);
  } catch (err) {
    grid.innerHTML = `<p class="orders-empty">${t('favorites.err_loading')}</p>`;
  }
}

function onPageLocaleChange() {
  loadFavorites();
}

function init() {
  if (!requireLogin()) return;
  updateHeaderForAuth();
  loadFavorites();
}

init();
