let categories = [];

const orderStatusLabels = {
  pending: 'Oczekujące',
  paid: 'Opłacone',
  processing: 'W realizacji',
  shipped: 'Wysłane',
  delivered: 'Dostarczone',
  cancelled: 'Anulowane',
  refunded: 'Zwrócone',
};

function showDashboardBanner(message, type = 'error') {
  const banner = document.getElementById('dashboardBanner');
  banner.textContent = message;
  banner.className = `dashboard-banner show ${type}`;
}

function clearDashboardBanner() {
  const banner = document.getElementById('dashboardBanner');
  banner.className = 'dashboard-banner';
  banner.textContent = '';
}

function requireLogin() {
  const user = getCurrentUser();
  if (!user) {
    window.location.href = 'login.html?redirect=dashboard.html';
    return null;
  }
  return user;
}

async function loadCategories() {
  try {
    const res = await fetch(`${API_URL}/categories`);
    categories = await res.json();
    const select = document.getElementById('listingCategory');
    select.innerHTML =
      '<option value="">Bez kategorii</option>' +
      categories.map((c) => `<option value="${c.id}">${escapeHtml(c.name)}</option>`).join('');
  } catch (err) {
    console.error('Nie udało się pobrać kategorii', err);
  }
}

function setPhotoPreview(previewId, url) {
  const img = document.getElementById(previewId);
  if (url) {
    img.src = url;
    img.style.display = 'block';
  } else {
    img.src = '';
    img.style.display = 'none';
  }
}

function bindPhotoUpload(fileInputId, hiddenInputId, previewId) {
  document.getElementById(fileInputId).addEventListener('change', async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    clearDashboardBanner();
    const url = await uploadPhoto(file);
    if (!url) {
      showDashboardBanner('Nie udało się wgrać zdjęcia.');
      return;
    }
    document.getElementById(hiddenInputId).value = url;
    setPhotoPreview(previewId, url);
  });
}

function fillEditShopForm(shop) {
  document.getElementById('editShopName').value = shop.name || '';
  document.getElementById('editShopDescription').value = shop.description || '';
  document.getElementById('editShopAvatarUrl').value = shop.avatar_url || '';
  document.getElementById('editShopBannerUrl').value = shop.banner_url || '';
  setPhotoPreview('editShopAvatarPreview', shop.avatar_url);
  setPhotoPreview('editShopBannerPreview', shop.banner_url);
  const link = document.getElementById('shopPublicLink');
  link.href = `shop-profile.html?id=${shop.id}`;
  link.textContent = `shop-profile.html?id=${shop.id}`;
}

async function loadMyShop() {
  const res = await authFetch('/shops/me');
  if (res.status === 404) {
    document.getElementById('noShopState').style.display = 'block';
    document.getElementById('dashboardMain').style.display = 'none';
    return null;
  }
  if (!res.ok) {
    showDashboardBanner('Nie udało się pobrać danych sklepu.');
    return null;
  }
  const shop = await res.json();
  document.getElementById('noShopState').style.display = 'none';
  document.getElementById('dashboardMain').style.display = 'block';
  fillEditShopForm(shop);
  loadStats();
  return shop;
}

async function loadStats() {
  const grid = document.getElementById('statsGrid');
  grid.innerHTML = '<p class="dashboard-empty">Ładowanie...</p>';
  try {
    const res = await authFetch('/seller/stats');
    const stats = await res.json();
    if (!res.ok) {
      grid.innerHTML = '<p class="dashboard-empty">Nie udało się pobrać statystyk.</p>';
      return;
    }
    grid.innerHTML = `
      <div class="dashboard-stat-tile">
        <p class="dashboard-stat-value">${stats.total_orders}</p>
        <p class="dashboard-stat-label">Zamówienia</p>
      </div>
      <div class="dashboard-stat-tile">
        <p class="dashboard-stat-value">${parseFloat(stats.total_revenue).toFixed(2)} PLN</p>
        <p class="dashboard-stat-label">Przychód</p>
      </div>
      <div class="dashboard-stat-tile">
        <p class="dashboard-stat-value">${stats.listings_active}/${stats.listings_total}</p>
        <p class="dashboard-stat-label">Aktywne oferty</p>
      </div>
    `;
  } catch (err) {
    grid.innerHTML = '<p class="dashboard-empty">Błąd ładowania statystyk.</p>';
  }
}

function switchTab(tabName) {
  document.querySelectorAll('.dashboard-tab').forEach((b) => b.classList.toggle('active', b.dataset.tab === tabName));
  document.querySelectorAll('.dashboard-tab-panel').forEach((p) => {
    p.style.display = p.id === `tab-${tabName}` ? 'block' : 'none';
  });
  if (tabName === 'listings') loadListings();
  if (tabName === 'orders') loadOrders();
}

function resetListingForm() {
  document.getElementById('listingId').value = '';
  document.getElementById('listingTitle').value = '';
  document.getElementById('listingDescription').value = '';
  document.getElementById('listingPrice').value = '';
  document.getElementById('listingQuantity').value = '1';
  document.getElementById('listingCategory').value = '';
  document.getElementById('listingPhotoUrl').value = '';
  document.getElementById('listingPhotoFile').value = '';
  setPhotoPreview('listingPhotoPreview', null);
  document.getElementById('listingSubmitBtn').textContent = 'Dodaj ofertę';
}

function editListing(item) {
  document.getElementById('listingId').value = item.id;
  document.getElementById('listingTitle').value = item.title;
  document.getElementById('listingDescription').value = item.description || '';
  document.getElementById('listingPrice').value = item.price;
  document.getElementById('listingQuantity').value = item.quantity;
  document.getElementById('listingCategory').value = item.category_id || '';
  document.getElementById('listingPhotoUrl').value = item.primary_photo || '';
  document.getElementById('listingPhotoFile').value = '';
  setPhotoPreview('listingPhotoPreview', item.primary_photo);
  document.getElementById('listingSubmitBtn').textContent = 'Zapisz zmiany';
  document.getElementById('listingForm').style.display = 'flex';
}

async function toggleListingActive(item) {
  clearDashboardBanner();
  try {
    let res;
    if (item.is_active) {
      res = await authFetch(`/listings/${item.id}`, { method: 'DELETE' });
    } else {
      res = await authFetch(`/listings/${item.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: item.title,
          description: item.description || '',
          category_id: item.category_id || null,
          price: item.price,
          currency: item.currency,
          quantity: item.quantity,
          is_active: true,
        }),
      });
    }
    const data = await res.json();
    if (!res.ok) {
      showDashboardBanner(data.message || 'Nie udało się zaktualizować oferty.');
      return;
    }
    loadListings();
  } catch (err) {
    showDashboardBanner('Nie udało się połączyć z serwerem.');
  }
}

async function loadListings() {
  const container = document.getElementById('listingsTable');
  container.innerHTML = '<p class="dashboard-empty">Ładowanie...</p>';
  try {
    const res = await authFetch('/seller/listings');
    const items = await res.json();
    if (!res.ok) {
      container.innerHTML = '<p class="dashboard-empty">Nie udało się pobrać ofert.</p>';
      return;
    }
    if (items.length === 0) {
      container.innerHTML = '<p class="dashboard-empty">Nie masz jeszcze żadnych ofert.</p>';
      return;
    }

    container.innerHTML = items
      .map(
        (item) => `
      <div class="dashboard-row" data-id="${item.id}">
        <img src="${item.primary_photo || 'https://picsum.photos/seed/placeholder/100/100'}" alt="${escapeHtml(item.title)}" />
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(item.title)}</p>
          <p class="dashboard-row-meta">${parseFloat(item.price).toFixed(2)} ${item.currency} · ${item.quantity} szt. · ${item.is_active ? 'Aktywna' : 'Nieaktywna'}</p>
        </div>
        <div class="dashboard-row-actions">
          <button class="dashboard-edit-btn" data-id="${item.id}">Edytuj</button>
          <button class="dashboard-toggle-btn" data-id="${item.id}">${item.is_active ? 'Wyłącz' : 'Włącz'}</button>
        </div>
      </div>
    `
      )
      .join('');

    container.querySelectorAll('.dashboard-edit-btn').forEach((btn) => {
      btn.addEventListener('click', () => {
        const item = items.find((i) => i.id === btn.dataset.id);
        if (item) editListing(item);
      });
    });
    container.querySelectorAll('.dashboard-toggle-btn').forEach((btn) => {
      btn.addEventListener('click', () => {
        const item = items.find((i) => i.id === btn.dataset.id);
        if (item) toggleListingActive(item);
      });
    });
  } catch (err) {
    container.innerHTML = '<p class="dashboard-empty">Błąd ładowania ofert.</p>';
  }
}

async function loadOrders() {
  const container = document.getElementById('ordersTable');
  container.innerHTML = '<p class="dashboard-empty">Ładowanie...</p>';
  try {
    const res = await authFetch('/seller/orders');
    const orders = await res.json();
    if (!res.ok) {
      container.innerHTML = '<p class="dashboard-empty">Nie udało się pobrać zamówień.</p>';
      return;
    }
    if (orders.length === 0) {
      container.innerHTML = '<p class="dashboard-empty">Brak zamówień.</p>';
      return;
    }

    container.innerHTML = orders
      .map(
        (o) => `
      <div class="dashboard-row" data-id="${o.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">Zamówienie #${o.id.slice(0, 8)}</p>
          <p class="dashboard-row-meta">Kupujący: ${escapeHtml(o.buyer_username || '—')} · ${parseFloat(o.total_amount).toFixed(2)} ${o.currency} · ${new Date(o.created_at).toLocaleDateString('pl-PL')}</p>
        </div>
        <select class="dashboard-order-status" data-id="${o.id}">
          ${Object.entries(orderStatusLabels)
            .map(([value, label]) => `<option value="${value}" ${value === o.status ? 'selected' : ''}>${label}</option>`)
            .join('')}
        </select>
      </div>
    `
      )
      .join('');

    container.querySelectorAll('.dashboard-order-status').forEach((select) => {
      select.addEventListener('change', async () => {
        clearDashboardBanner();
        try {
          const res = await authFetch(`/seller/orders/${select.dataset.id}/status`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ status: select.value }),
          });
          const data = await res.json();
          if (!res.ok) {
            showDashboardBanner(data.message || 'Nie udało się zaktualizować statusu.');
          }
        } catch (err) {
          showDashboardBanner('Nie udało się połączyć z serwerem.');
        }
      });
    });
  } catch (err) {
    container.innerHTML = '<p class="dashboard-empty">Błąd ładowania zamówień.</p>';
  }
}

function bindEvents() {
  bindPhotoUpload('shopAvatarFile', 'shopAvatarUrl', 'shopAvatarPreview');
  bindPhotoUpload('shopBannerFile', 'shopBannerUrl', 'shopBannerPreview');
  bindPhotoUpload('editShopAvatarFile', 'editShopAvatarUrl', 'editShopAvatarPreview');
  bindPhotoUpload('editShopBannerFile', 'editShopBannerUrl', 'editShopBannerPreview');
  bindPhotoUpload('listingPhotoFile', 'listingPhotoUrl', 'listingPhotoPreview');

  document.getElementById('createShopForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    clearDashboardBanner();
    const payload = {
      name: document.getElementById('shopName').value.trim(),
      description: document.getElementById('shopDescription').value.trim(),
      avatar_url: document.getElementById('shopAvatarUrl').value.trim(),
      banner_url: document.getElementById('shopBannerUrl').value.trim(),
    };
    try {
      const res = await authFetch('/shops', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (!res.ok) {
        showDashboardBanner(data.message || 'Nie udało się założyć sklepu.');
        return;
      }

      const storage = localStorage.getItem('user') ? localStorage : sessionStorage;
      const user = JSON.parse(storage.getItem('user'));
      user.is_seller = true;
      storage.setItem('user', JSON.stringify(user));
      updateHeaderForAuth();

      document.getElementById('noShopState').style.display = 'none';
      document.getElementById('dashboardMain').style.display = 'block';
      fillEditShopForm(data);
      switchTab('shop');
    } catch (err) {
      showDashboardBanner('Nie udało się połączyć z serwerem.');
    }
  });

  document.getElementById('editShopForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    clearDashboardBanner();
    const payload = {
      name: document.getElementById('editShopName').value.trim(),
      description: document.getElementById('editShopDescription').value.trim(),
      avatar_url: document.getElementById('editShopAvatarUrl').value.trim(),
      banner_url: document.getElementById('editShopBannerUrl').value.trim(),
    };
    try {
      const res = await authFetch('/shops/me', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (!res.ok) {
        showDashboardBanner(data.message || 'Nie udało się zapisać zmian.');
        return;
      }
      showDashboardBanner('Zapisano zmiany.', 'success');
    } catch (err) {
      showDashboardBanner('Nie udało się połączyć z serwerem.');
    }
  });

  document.querySelectorAll('.dashboard-tab').forEach((btn) => {
    btn.addEventListener('click', () => switchTab(btn.dataset.tab));
  });

  document.getElementById('newListingBtn').addEventListener('click', () => {
    resetListingForm();
    document.getElementById('listingForm').style.display = 'flex';
  });

  document.getElementById('cancelListingBtn').addEventListener('click', () => {
    document.getElementById('listingForm').style.display = 'none';
    resetListingForm();
  });

  document.getElementById('listingForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    clearDashboardBanner();
    const id = document.getElementById('listingId').value;
    const photoUrl = document.getElementById('listingPhotoUrl').value.trim();
    const payload = {
      title: document.getElementById('listingTitle').value.trim(),
      description: document.getElementById('listingDescription').value.trim(),
      category_id: document.getElementById('listingCategory').value || null,
      price: document.getElementById('listingPrice').value,
      currency: 'PLN',
      quantity: parseInt(document.getElementById('listingQuantity').value, 10) || 0,
      photos: photoUrl ? [photoUrl] : [],
    };

    try {
      const res = await authFetch(id ? `/listings/${id}` : '/listings', {
        method: id ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (!res.ok) {
        showDashboardBanner(data.message || 'Nie udało się zapisać oferty.');
        return;
      }
      document.getElementById('listingForm').style.display = 'none';
      resetListingForm();
      loadListings();
    } catch (err) {
      showDashboardBanner('Nie udało się połączyć z serwerem.');
    }
  });
}

async function init() {
  if (!requireLogin()) return;
  updateHeaderForAuth();
  bindEvents();
  await loadCategories();
  await loadMyShop();
}

init();
