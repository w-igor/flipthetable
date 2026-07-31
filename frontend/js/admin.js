const orderStatusLabels = {
  pending: 'Oczekujące',
  paid: 'Opłacone',
  processing: 'W realizacji',
  shipped: 'Wysłane',
  delivered: 'Dostarczone',
  cancelled: 'Anulowane',
  refunded: 'Zwrócone',
};

const auditActionLabels = {
  'user.activate': 'Odblokowano użytkownika',
  'user.deactivate': 'Zablokowano użytkownika',
  'user.grant_admin': 'Nadano uprawnienia administratora',
  'user.revoke_admin': 'Odebrano uprawnienia administratora',
  'shop.activate': 'Odblokowano sklep',
  'shop.deactivate': 'Zablokowano sklep',
  'listing.activate': 'Przywrócono ofertę',
  'listing.deactivate': 'Ukryto ofertę',
  'category.create': 'Utworzono kategorię',
  'category.update': 'Zaktualizowano kategorię',
  'category.delete': 'Usunięto kategorię',
};

const state = {
  users: { page: 1, search: '' },
  shops: { page: 1, search: '' },
  listings: { page: 1, search: '', active: '' },
  orders: { page: 1, status: '' },
  audit: { page: 1 },
};

let categoriesCache = [];

function showAdminBanner(message, type = 'error') {
  const banner = document.getElementById('adminBanner');
  banner.textContent = message;
  banner.className = `dashboard-banner show ${type}`;
}

function clearAdminBanner() {
  const banner = document.getElementById('adminBanner');
  banner.className = 'dashboard-banner';
  banner.textContent = '';
}

function requireLogin() {
  const user = getCurrentUser();
  if (!user) {
    window.location.href = 'login.html?redirect=admin.html';
    return null;
  }
  return user;
}

function renderPager(containerId, meta, onPage) {
  const el = document.getElementById(containerId);
  if (meta.total_pages <= 1) {
    el.innerHTML = '';
    return;
  }
  el.innerHTML = `
    <button id="${containerId}-prev" ${meta.page <= 1 ? 'disabled' : ''}>&larr; Poprzednia</button>
    <span>Strona ${meta.page} z ${meta.total_pages} (${meta.total} wyników)</span>
    <button id="${containerId}-next" ${meta.page >= meta.total_pages ? 'disabled' : ''}>Następna &rarr;</button>
  `;
  document.getElementById(`${containerId}-prev`)?.addEventListener('click', () => onPage(meta.page - 1));
  document.getElementById(`${containerId}-next`)?.addEventListener('click', () => onPage(meta.page + 1));
}

function renderOverviewFromStats(s) {
  const statsGrid = document.getElementById('overviewStats');
  const statusGrid = document.getElementById('overviewOrderStatus');
  statsGrid.innerHTML = `
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.total_users}</p><p class="dashboard-stat-label">Użytkownicy (${s.active_users} aktywnych)</p></div>
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.total_sellers}</p><p class="dashboard-stat-label">Sprzedawcy</p></div>
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.total_shops}</p><p class="dashboard-stat-label">Sklepy (${s.active_shops} aktywnych)</p></div>
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.total_listings}</p><p class="dashboard-stat-label">Oferty (${s.active_listings} aktywnych)</p></div>
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.total_orders}</p><p class="dashboard-stat-label">Zamówienia</p></div>
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${parseFloat(s.total_revenue).toFixed(2)} PLN</p><p class="dashboard-stat-label">Przychód (bez anulowanych/zwróconych)</p></div>
  `;
  statusGrid.innerHTML = Object.entries(orderStatusLabels)
    .map(([key, label]) => `
      <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.orders_by_status[key] || 0}</p><p class="dashboard-stat-label">${label}</p></div>
    `)
    .join('');
}

async function loadOverview() {
  const statsGrid = document.getElementById('overviewStats');
  statsGrid.innerHTML = '<p class="dashboard-empty">Ładowanie...</p>';
  try {
    const res = await authFetch('/admin/stats');
    const s = await res.json();
    if (!res.ok) {
      statsGrid.innerHTML = '<p class="dashboard-empty">Nie udało się pobrać statystyk.</p>';
      return;
    }
    renderOverviewFromStats(s);
  } catch (err) {
    statsGrid.innerHTML = '<p class="dashboard-empty">Błąd ładowania statystyk.</p>';
  }
}

async function loadUsers() {
  const container = document.getElementById('usersTable');
  container.innerHTML = '<p class="dashboard-empty">Ładowanie...</p>';
  try {
    const params = new URLSearchParams({ page: state.users.page, page_size: 20 });
    if (state.users.search) params.set('q', state.users.search);
    const res = await authFetch(`/admin/users?${params}`);
    const data = await res.json();
    if (!res.ok) {
      container.innerHTML = '<p class="dashboard-empty">Nie udało się pobrać użytkowników.</p>';
      return;
    }
    if (data.items.length === 0) {
      container.innerHTML = '<p class="dashboard-empty">Brak wyników.</p>';
      renderPager('usersPager', data, () => {});
      return;
    }
    const me = getCurrentUser();
    container.innerHTML = data.items
      .map(
        (u) => `
      <div class="dashboard-row" data-id="${u.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(u.username)} ${u.is_admin ? '<span class="admin-badge admin">Admin</span>' : ''} <span class="admin-badge ${u.is_active ? 'active' : 'inactive'}">${u.is_active ? 'Aktywny' : 'Zablokowany'}</span></p>
          <p class="dashboard-row-meta">${escapeHtml(u.email)} · ${u.is_seller ? 'Sprzedawca' : 'Kupujący'} · dołączył ${new Date(u.created_at).toLocaleDateString('pl-PL')}</p>
        </div>
        <div class="dashboard-row-actions">
          ${
            u.id === me?.id
              ? ''
              : `<button class="dashboard-edit-btn admin-toggle-admin-btn" data-id="${u.id}" data-is-admin="${u.is_admin}" data-username="${escapeHtml(u.username)}">${u.is_admin ? 'Odbierz admina' : 'Nadaj admina'}</button>`
          }
          <button class="dashboard-toggle-btn" data-id="${u.id}" data-active="${u.is_active}">${u.is_active ? 'Zablokuj' : 'Odblokuj'}</button>
        </div>
      </div>
    `
      )
      .join('');

    container.querySelectorAll('.dashboard-toggle-btn').forEach((btn) => {
      btn.addEventListener('click', async () => {
        clearAdminBanner();
        const nextActive = btn.dataset.active !== 'true';
        try {
          const res = await authFetch(`/admin/users/${btn.dataset.id}/status`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ is_active: nextActive }),
          });
          const d = await res.json();
          if (!res.ok) {
            showAdminBanner(d.message || 'Nie udało się zaktualizować użytkownika.');
            return;
          }
          loadUsers();
        } catch (err) {
          showAdminBanner('Nie udało się połączyć z serwerem.');
        }
      });
    });

    container.querySelectorAll('.admin-toggle-admin-btn').forEach((btn) => {
      btn.addEventListener('click', async () => {
        clearAdminBanner();
        const nextIsAdmin = btn.dataset.isAdmin !== 'true';
        const verb = nextIsAdmin ? 'nadać uprawnienia administratora' : 'odebrać uprawnienia administratora';
        if (!confirm(`Czy na pewno chcesz ${verb} użytkownikowi "${btn.dataset.username}"?`)) return;
        try {
          const res = await authFetch(`/admin/users/${btn.dataset.id}/admin-status`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ is_admin: nextIsAdmin }),
          });
          const d = await res.json();
          if (!res.ok) {
            showAdminBanner(d.message || 'Nie udało się zaktualizować uprawnień.');
            return;
          }
          loadUsers();
        } catch (err) {
          showAdminBanner('Nie udało się połączyć z serwerem.');
        }
      });
    });

    renderPager('usersPager', data, (page) => {
      state.users.page = page;
      loadUsers();
    });
  } catch (err) {
    container.innerHTML = '<p class="dashboard-empty">Błąd ładowania użytkowników.</p>';
  }
}

async function loadShops() {
  const container = document.getElementById('shopsTable');
  container.innerHTML = '<p class="dashboard-empty">Ładowanie...</p>';
  try {
    const params = new URLSearchParams({ page: state.shops.page, page_size: 20 });
    if (state.shops.search) params.set('q', state.shops.search);
    const res = await authFetch(`/admin/shops?${params}`);
    const data = await res.json();
    if (!res.ok) {
      container.innerHTML = '<p class="dashboard-empty">Nie udało się pobrać sklepów.</p>';
      return;
    }
    if (data.items.length === 0) {
      container.innerHTML = '<p class="dashboard-empty">Brak wyników.</p>';
      renderPager('shopsPager', data, () => {});
      return;
    }
    container.innerHTML = data.items
      .map(
        (s) => `
      <div class="dashboard-row" data-id="${s.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(s.name)} <span class="admin-badge ${s.is_active ? 'active' : 'inactive'}">${s.is_active ? 'Aktywny' : 'Zablokowany'}</span></p>
          <p class="dashboard-row-meta">Właściciel: ${escapeHtml(s.owner_username)} · ${s.listings_count} ofert · ${s.sales_count} sprzedaży</p>
        </div>
        <div class="dashboard-row-actions">
          <a href="shop-profile.html?id=${s.id}" target="_blank" class="dashboard-edit-btn">Podgląd</a>
          <button class="dashboard-toggle-btn" data-id="${s.id}" data-active="${s.is_active}">${s.is_active ? 'Zablokuj' : 'Odblokuj'}</button>
        </div>
      </div>
    `
      )
      .join('');

    container.querySelectorAll('.dashboard-toggle-btn').forEach((btn) => {
      btn.addEventListener('click', async () => {
        clearAdminBanner();
        const nextActive = btn.dataset.active !== 'true';
        try {
          const res = await authFetch(`/admin/shops/${btn.dataset.id}/status`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ is_active: nextActive }),
          });
          const d = await res.json();
          if (!res.ok) {
            showAdminBanner(d.message || 'Nie udało się zaktualizować sklepu.');
            return;
          }
          loadShops();
        } catch (err) {
          showAdminBanner('Nie udało się połączyć z serwerem.');
        }
      });
    });

    renderPager('shopsPager', data, (page) => {
      state.shops.page = page;
      loadShops();
    });
  } catch (err) {
    container.innerHTML = '<p class="dashboard-empty">Błąd ładowania sklepów.</p>';
  }
}

async function loadListings() {
  const container = document.getElementById('listingsTable');
  container.innerHTML = '<p class="dashboard-empty">Ładowanie...</p>';
  try {
    const params = new URLSearchParams({ page: state.listings.page, page_size: 20 });
    if (state.listings.search) params.set('q', state.listings.search);
    if (state.listings.active) params.set('active', state.listings.active);
    const res = await authFetch(`/admin/listings?${params}`);
    const data = await res.json();
    if (!res.ok) {
      container.innerHTML = '<p class="dashboard-empty">Nie udało się pobrać ofert.</p>';
      return;
    }
    if (data.items.length === 0) {
      container.innerHTML = '<p class="dashboard-empty">Brak wyników.</p>';
      renderPager('listingsPager', data, () => {});
      return;
    }
    container.innerHTML = data.items
      .map(
        (l) => `
      <div class="dashboard-row" data-id="${l.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(l.title)} <span class="admin-badge ${l.is_active ? 'active' : 'inactive'}">${l.is_active ? 'Aktywna' : 'Nieaktywna'}</span></p>
          <p class="dashboard-row-meta">${parseFloat(l.price).toFixed(2)} ${l.currency} · sklep: ${escapeHtml(l.shop_name)} (${escapeHtml(l.seller_username)}) · ${l.quantity} szt. · ${l.sales_count} sprzedaży</p>
        </div>
        <div class="dashboard-row-actions">
          <a href="listing.html?id=${l.id}" target="_blank" class="dashboard-edit-btn">Podgląd</a>
          <button class="dashboard-toggle-btn" data-id="${l.id}" data-active="${l.is_active}">${l.is_active ? 'Ukryj' : 'Przywróć'}</button>
        </div>
      </div>
    `
      )
      .join('');

    container.querySelectorAll('.dashboard-toggle-btn').forEach((btn) => {
      btn.addEventListener('click', async () => {
        clearAdminBanner();
        const nextActive = btn.dataset.active !== 'true';
        try {
          const res = await authFetch(`/admin/listings/${btn.dataset.id}/status`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ is_active: nextActive }),
          });
          const d = await res.json();
          if (!res.ok) {
            showAdminBanner(d.message || 'Nie udało się zaktualizować oferty.');
            return;
          }
          loadListings();
        } catch (err) {
          showAdminBanner('Nie udało się połączyć z serwerem.');
        }
      });
    });

    renderPager('listingsPager', data, (page) => {
      state.listings.page = page;
      loadListings();
    });
  } catch (err) {
    container.innerHTML = '<p class="dashboard-empty">Błąd ładowania ofert.</p>';
  }
}

async function loadOrders() {
  const container = document.getElementById('ordersTable');
  container.innerHTML = '<p class="dashboard-empty">Ładowanie...</p>';
  try {
    const params = new URLSearchParams({ page: state.orders.page, page_size: 20 });
    if (state.orders.status) params.set('status', state.orders.status);
    const res = await authFetch(`/admin/orders?${params}`);
    const data = await res.json();
    if (!res.ok) {
      container.innerHTML = '<p class="dashboard-empty">Nie udało się pobrać zamówień.</p>';
      return;
    }
    if (data.items.length === 0) {
      container.innerHTML = '<p class="dashboard-empty">Brak wyników.</p>';
      renderPager('ordersPager', data, () => {});
      return;
    }
    container.innerHTML = data.items
      .map(
        (o) => `
      <div class="dashboard-row" data-id="${o.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">Zamówienie #${o.id.slice(0, 8)} · ${orderStatusLabels[o.status] || o.status}</p>
          <p class="dashboard-row-meta">Kupujący: ${escapeHtml(o.buyer_username || '—')} · Sklep: ${escapeHtml(o.shop_name)} · ${parseFloat(o.total_amount).toFixed(2)} ${o.currency} · ${new Date(o.created_at).toLocaleDateString('pl-PL')} · Płatność: ${o.payment_status === 'completed' ? 'opłacone' : o.payment_status === 'failed' ? 'odrzucona' : 'oczekuje'}</p>
        </div>
      </div>
    `
      )
      .join('');

    renderPager('ordersPager', data, (page) => {
      state.orders.page = page;
      loadOrders();
    });
  } catch (err) {
    container.innerHTML = '<p class="dashboard-empty">Błąd ładowania zamówień.</p>';
  }
}

function categoryPathName(cat, byId) {
  if (!cat.parent_id) return cat.name;
  const parent = byId[cat.parent_id];
  return parent ? `${parent.name} / ${cat.name}` : cat.name;
}

function populateCategoryParentSelect(excludeId) {
  const select = document.getElementById('categoryParent');
  select.innerHTML =
    '<option value="">Brak (kategoria główna)</option>' +
    categoriesCache
      .filter((c) => c.id !== excludeId)
      .map((c) => `<option value="${c.id}">${escapeHtml(c.name)}</option>`)
      .join('');
}

function resetCategoryForm() {
  document.getElementById('categoryId').value = '';
  document.getElementById('categoryName').value = '';
  document.getElementById('categoryDescription').value = '';
  document.getElementById('categorySortOrder').value = '0';
  populateCategoryParentSelect(null);
  document.getElementById('categoryParent').value = '';
  document.getElementById('categorySubmitBtn').textContent = 'Dodaj kategorię';
}

function editCategory(cat) {
  document.getElementById('categoryId').value = cat.id;
  document.getElementById('categoryName').value = cat.name;
  document.getElementById('categoryDescription').value = cat.description || '';
  document.getElementById('categorySortOrder').value = cat.sort_order;
  populateCategoryParentSelect(cat.id);
  document.getElementById('categoryParent').value = cat.parent_id || '';
  document.getElementById('categorySubmitBtn').textContent = 'Zapisz zmiany';
  document.getElementById('categoryForm').style.display = 'flex';
}

async function loadCategories() {
  const container = document.getElementById('categoriesTable');
  container.innerHTML = '<p class="dashboard-empty">Ładowanie...</p>';
  try {
    const res = await authFetch('/admin/categories');
    const items = await res.json();
    if (!res.ok) {
      container.innerHTML = '<p class="dashboard-empty">Nie udało się pobrać kategorii.</p>';
      return;
    }
    categoriesCache = items;
    const byId = Object.fromEntries(items.map((c) => [c.id, c]));

    if (items.length === 0) {
      container.innerHTML = '<p class="dashboard-empty">Brak kategorii. Dodaj pierwszą.</p>';
      return;
    }

    container.innerHTML = items
      .map(
        (c) => `
      <div class="dashboard-row" data-id="${c.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(categoryPathName(c, byId))}</p>
          <p class="dashboard-row-meta">slug: ${escapeHtml(c.slug)} · kolejność: ${c.sort_order}${c.description ? ' · ' + escapeHtml(c.description) : ''}</p>
        </div>
        <div class="dashboard-row-actions">
          <button class="dashboard-edit-btn" data-id="${c.id}">Edytuj</button>
          <button class="dashboard-toggle-btn" data-id="${c.id}">Usuń</button>
        </div>
      </div>
    `
      )
      .join('');

    container.querySelectorAll('.dashboard-edit-btn').forEach((btn) => {
      btn.addEventListener('click', () => {
        const cat = items.find((c) => c.id === btn.dataset.id);
        if (cat) editCategory(cat);
      });
    });
    container.querySelectorAll('.dashboard-toggle-btn').forEach((btn) => {
      btn.addEventListener('click', async () => {
        const cat = items.find((c) => c.id === btn.dataset.id);
        if (!cat) return;
        if (!confirm(`Usunąć kategorię "${cat.name}"? Podkategorie staną się głównymi, a oferty stracą przypisanie.`)) return;
        clearAdminBanner();
        try {
          const res = await authFetch(`/admin/categories/${cat.id}`, { method: 'DELETE' });
          const d = await res.json();
          if (!res.ok) {
            showAdminBanner(d.message || 'Nie udało się usunąć kategorii.');
            return;
          }
          loadCategories();
        } catch (err) {
          showAdminBanner('Nie udało się połączyć z serwerem.');
        }
      });
    });
  } catch (err) {
    container.innerHTML = '<p class="dashboard-empty">Błąd ładowania kategorii.</p>';
  }
}

async function loadAuditLog() {
  const container = document.getElementById('auditTable');
  container.innerHTML = '<p class="dashboard-empty">Ładowanie...</p>';
  try {
    const params = new URLSearchParams({ page: state.audit.page, page_size: 30 });
    const res = await authFetch(`/admin/audit-log?${params}`);
    const data = await res.json();
    if (!res.ok) {
      container.innerHTML = '<p class="dashboard-empty">Nie udało się pobrać dziennika.</p>';
      return;
    }
    if (data.items.length === 0) {
      container.innerHTML = '<p class="dashboard-empty">Brak wpisów.</p>';
      renderPager('auditPager', data, () => {});
      return;
    }
    container.innerHTML = data.items
      .map(
        (e) => `
      <div class="dashboard-row" data-id="${e.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${auditActionLabels[e.action] || e.action}${e.details ? ': ' + escapeHtml(e.details) : ''}</p>
          <p class="dashboard-row-meta">${escapeHtml(e.admin_username)} · ${new Date(e.created_at).toLocaleString('pl-PL')}</p>
        </div>
      </div>
    `
      )
      .join('');

    renderPager('auditPager', data, (page) => {
      state.audit.page = page;
      loadAuditLog();
    });
  } catch (err) {
    container.innerHTML = '<p class="dashboard-empty">Błąd ładowania dziennika.</p>';
  }
}

function switchTab(tabName) {
  document.querySelectorAll('.dashboard-tab').forEach((b) => b.classList.toggle('active', b.dataset.tab === tabName));
  document.querySelectorAll('.dashboard-tab-panel').forEach((p) => {
    p.style.display = p.id === `tab-${tabName}` ? 'block' : 'none';
  });
  if (tabName === 'overview') loadOverview();
  if (tabName === 'users') loadUsers();
  if (tabName === 'shops') loadShops();
  if (tabName === 'listings') loadListings();
  if (tabName === 'orders') loadOrders();
  if (tabName === 'categories') loadCategories();
  if (tabName === 'audit') loadAuditLog();
}

function debounce(fn, delay) {
  let timer;
  return (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => fn(...args), delay);
  };
}

function bindEvents() {
  document.querySelectorAll('.dashboard-tab').forEach((btn) => {
    btn.addEventListener('click', () => switchTab(btn.dataset.tab));
  });

  document.getElementById('usersSearch').addEventListener(
    'input',
    debounce((e) => {
      state.users.search = e.target.value.trim();
      state.users.page = 1;
      loadUsers();
    }, 350)
  );

  document.getElementById('shopsSearch').addEventListener(
    'input',
    debounce((e) => {
      state.shops.search = e.target.value.trim();
      state.shops.page = 1;
      loadShops();
    }, 350)
  );

  document.getElementById('listingsSearch').addEventListener(
    'input',
    debounce((e) => {
      state.listings.search = e.target.value.trim();
      state.listings.page = 1;
      loadListings();
    }, 350)
  );

  document.getElementById('listingsActiveFilter').addEventListener('change', (e) => {
    state.listings.active = e.target.value;
    state.listings.page = 1;
    loadListings();
  });

  document.getElementById('ordersStatusFilter').addEventListener('change', (e) => {
    state.orders.status = e.target.value;
    state.orders.page = 1;
    loadOrders();
  });

  document.getElementById('newCategoryBtn').addEventListener('click', () => {
    resetCategoryForm();
    document.getElementById('categoryForm').style.display = 'flex';
  });

  document.getElementById('cancelCategoryBtn').addEventListener('click', () => {
    document.getElementById('categoryForm').style.display = 'none';
    resetCategoryForm();
  });

  document.getElementById('categoryForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    clearAdminBanner();
    const id = document.getElementById('categoryId').value;
    const payload = {
      name: document.getElementById('categoryName').value.trim(),
      parent_id: document.getElementById('categoryParent').value || null,
      description: document.getElementById('categoryDescription').value.trim(),
      sort_order: parseInt(document.getElementById('categorySortOrder').value, 10) || 0,
    };
    try {
      const res = await authFetch(id ? `/admin/categories/${id}` : '/admin/categories', {
        method: id ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (!res.ok) {
        showAdminBanner(data.message || 'Nie udało się zapisać kategorii.');
        return;
      }
      document.getElementById('categoryForm').style.display = 'none';
      resetCategoryForm();
      loadCategories();
    } catch (err) {
      showAdminBanner('Nie udało się połączyć z serwerem.');
    }
  });
}

async function init() {
  if (!requireLogin()) return;
  updateHeaderForAuth();
  bindEvents();

  try {
    const res = await authFetch('/admin/stats');
    if (res.status === 403) {
      document.getElementById('adminDenied').style.display = 'block';
      return;
    }
    if (!res.ok) {
      showAdminBanner('Nie udało się załadować panelu administratora.');
      return;
    }
    document.getElementById('adminMain').style.display = 'flex';
    const s = await res.json();
    renderOverviewFromStats(s);
  } catch (err) {
    showAdminBanner('Nie udało się połączyć z serwerem.');
  }
}

init();
