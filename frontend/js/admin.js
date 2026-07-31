const orderStatusValues = ['pending', 'paid', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded'];

const auditActionKeys = {
  'user.activate': 'audit_action.user_activate',
  'user.deactivate': 'audit_action.user_deactivate',
  'user.grant_admin': 'audit_action.user_grant_admin',
  'user.revoke_admin': 'audit_action.user_revoke_admin',
  'shop.activate': 'audit_action.shop_activate',
  'shop.deactivate': 'audit_action.shop_deactivate',
  'listing.activate': 'audit_action.listing_activate',
  'listing.deactivate': 'audit_action.listing_deactivate',
  'category.create': 'audit_action.category_create',
  'category.update': 'audit_action.category_update',
  'category.delete': 'audit_action.category_delete',
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
    <button id="${containerId}-prev" ${meta.page <= 1 ? 'disabled' : ''}>${t('admin.pager_prev')}</button>
    <span>${t('admin.pager_page_info', { page: meta.page, totalPages: meta.total_pages, total: meta.total })}</span>
    <button id="${containerId}-next" ${meta.page >= meta.total_pages ? 'disabled' : ''}>${t('admin.pager_next')}</button>
  `;
  document.getElementById(`${containerId}-prev`)?.addEventListener('click', () => onPage(meta.page - 1));
  document.getElementById(`${containerId}-next`)?.addEventListener('click', () => onPage(meta.page + 1));
}

function renderOverviewFromStats(s) {
  const statsGrid = document.getElementById('overviewStats');
  const statusGrid = document.getElementById('overviewOrderStatus');
  statsGrid.innerHTML = `
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.total_users}</p><p class="dashboard-stat-label">${t('admin.stat_users', { active: s.active_users })}</p></div>
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.total_sellers}</p><p class="dashboard-stat-label">${t('admin.stat_sellers')}</p></div>
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.total_shops}</p><p class="dashboard-stat-label">${t('admin.stat_shops', { active: s.active_shops })}</p></div>
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.total_listings}</p><p class="dashboard-stat-label">${t('admin.stat_listings', { active: s.active_listings })}</p></div>
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.total_orders}</p><p class="dashboard-stat-label">${t('admin.stat_orders')}</p></div>
    <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${parseFloat(s.total_revenue).toFixed(2)} PLN</p><p class="dashboard-stat-label">${t('admin.stat_revenue')}</p></div>
  `;
  statusGrid.innerHTML = orderStatusValues
    .map((key) => `
      <div class="dashboard-stat-tile"><p class="dashboard-stat-value">${s.orders_by_status[key] || 0}</p><p class="dashboard-stat-label">${t('order_status.' + key)}</p></div>
    `)
    .join('');
}

async function loadOverview() {
  const statsGrid = document.getElementById('overviewStats');
  statsGrid.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const res = await authFetch('/admin/stats');
    const s = await res.json();
    if (!res.ok) {
      statsGrid.innerHTML = `<p class="dashboard-empty">${t('admin.err_fetch_stats')}</p>`;
      return;
    }
    renderOverviewFromStats(s);
  } catch (err) {
    statsGrid.innerHTML = `<p class="dashboard-empty">${t('admin.err_loading_stats')}</p>`;
  }
}

async function loadUsers() {
  const container = document.getElementById('usersTable');
  container.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const params = new URLSearchParams({ page: state.users.page, page_size: 20 });
    if (state.users.search) params.set('q', state.users.search);
    const res = await authFetch(`/admin/users?${params}`);
    const data = await res.json();
    if (!res.ok) {
      container.innerHTML = `<p class="dashboard-empty">${t('admin.err_fetch_users')}</p>`;
      return;
    }
    if (data.items.length === 0) {
      container.innerHTML = `<p class="dashboard-empty">${t('admin.no_results')}</p>`;
      renderPager('usersPager', data, () => {});
      return;
    }
    const me = getCurrentUser();
    container.innerHTML = data.items
      .map(
        (u) => `
      <div class="dashboard-row" data-id="${u.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(u.username)} ${u.is_admin ? '<span class="admin-badge admin">Admin</span>' : ''} <span class="admin-badge ${u.is_active ? 'active' : 'inactive'}">${u.is_active ? t('admin.user_active_badge') : t('admin.user_blocked_badge')}</span></p>
          <p class="dashboard-row-meta">${escapeHtml(u.email)} · ${u.is_seller ? t('admin.role_seller') : t('admin.role_buyer')} · ${t('admin.joined_on', { date: new Date(u.created_at).toLocaleDateString('pl-PL') })}</p>
        </div>
        <div class="dashboard-row-actions">
          ${
            u.id === me?.id
              ? ''
              : `<button class="dashboard-edit-btn admin-toggle-admin-btn" data-id="${u.id}" data-is-admin="${u.is_admin}" data-username="${escapeHtml(u.username)}">${u.is_admin ? t('admin.revoke_admin_btn') : t('admin.grant_admin_btn')}</button>`
          }
          <button class="dashboard-toggle-btn" data-id="${u.id}" data-active="${u.is_active}">${u.is_active ? t('admin.block_btn') : t('admin.unblock_btn')}</button>
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
            showAdminBanner(d.message || t('admin.err_update_user'));
            return;
          }
          loadUsers();
        } catch (err) {
          showAdminBanner(t('common.error_connect'));
        }
      });
    });

    container.querySelectorAll('.admin-toggle-admin-btn').forEach((btn) => {
      btn.addEventListener('click', async () => {
        clearAdminBanner();
        const nextIsAdmin = btn.dataset.isAdmin !== 'true';
        const verb = nextIsAdmin ? t('admin.confirm_grant_admin') : t('admin.confirm_revoke_admin');
        if (!confirm(t('admin.confirm_admin_action', { verb, username: btn.dataset.username }))) return;
        try {
          const res = await authFetch(`/admin/users/${btn.dataset.id}/admin-status`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ is_admin: nextIsAdmin }),
          });
          const d = await res.json();
          if (!res.ok) {
            showAdminBanner(d.message || t('admin.err_update_permissions'));
            return;
          }
          loadUsers();
        } catch (err) {
          showAdminBanner(t('common.error_connect'));
        }
      });
    });

    renderPager('usersPager', data, (page) => {
      state.users.page = page;
      loadUsers();
    });
  } catch (err) {
    container.innerHTML = `<p class="dashboard-empty">${t('admin.err_loading_users')}</p>`;
  }
}

async function loadShops() {
  const container = document.getElementById('shopsTable');
  container.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const params = new URLSearchParams({ page: state.shops.page, page_size: 20 });
    if (state.shops.search) params.set('q', state.shops.search);
    const res = await authFetch(`/admin/shops?${params}`);
    const data = await res.json();
    if (!res.ok) {
      container.innerHTML = `<p class="dashboard-empty">${t('admin.err_fetch_shops')}</p>`;
      return;
    }
    if (data.items.length === 0) {
      container.innerHTML = `<p class="dashboard-empty">${t('admin.no_results')}</p>`;
      renderPager('shopsPager', data, () => {});
      return;
    }
    container.innerHTML = data.items
      .map(
        (s) => `
      <div class="dashboard-row" data-id="${s.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(s.name)} <span class="admin-badge ${s.is_active ? 'active' : 'inactive'}">${s.is_active ? t('admin.user_active_badge') : t('admin.user_blocked_badge')}</span></p>
          <p class="dashboard-row-meta">${t('admin.owner_label', { name: escapeHtml(s.owner_username) })} · ${t('admin.listings_sales_meta', { listings: s.listings_count, sales: s.sales_count })}</p>
        </div>
        <div class="dashboard-row-actions">
          <a href="shop-profile.html?id=${s.id}" target="_blank" class="dashboard-edit-btn">${t('admin.preview_btn')}</a>
          <button class="dashboard-toggle-btn" data-id="${s.id}" data-active="${s.is_active}">${s.is_active ? t('admin.block_btn') : t('admin.unblock_btn')}</button>
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
            showAdminBanner(d.message || t('admin.err_update_shop'));
            return;
          }
          loadShops();
        } catch (err) {
          showAdminBanner(t('common.error_connect'));
        }
      });
    });

    renderPager('shopsPager', data, (page) => {
      state.shops.page = page;
      loadShops();
    });
  } catch (err) {
    container.innerHTML = `<p class="dashboard-empty">${t('admin.err_loading_shops')}</p>`;
  }
}

async function loadListings() {
  const container = document.getElementById('listingsTable');
  container.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const params = new URLSearchParams({ page: state.listings.page, page_size: 20 });
    if (state.listings.search) params.set('q', state.listings.search);
    if (state.listings.active) params.set('active', state.listings.active);
    const res = await authFetch(`/admin/listings?${params}`);
    const data = await res.json();
    if (!res.ok) {
      container.innerHTML = `<p class="dashboard-empty">${t('dashboard.err_fetch_listings')}</p>`;
      return;
    }
    if (data.items.length === 0) {
      container.innerHTML = `<p class="dashboard-empty">${t('admin.no_results')}</p>`;
      renderPager('listingsPager', data, () => {});
      return;
    }
    container.innerHTML = data.items
      .map(
        (l) => `
      <div class="dashboard-row" data-id="${l.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(l.title)} <span class="admin-badge ${l.is_active ? 'active' : 'inactive'}">${l.is_active ? t('dashboard.status_active_f') : t('dashboard.status_inactive_f')}</span></p>
          <p class="dashboard-row-meta">${parseFloat(l.price).toFixed(2)} ${l.currency} · ${t('admin.shop_meta', { shop: escapeHtml(l.shop_name), seller: escapeHtml(l.seller_username), qty: l.quantity, sales: l.sales_count })}</p>
        </div>
        <div class="dashboard-row-actions">
          <a href="listing.html?id=${l.id}" target="_blank" class="dashboard-edit-btn">${t('admin.preview_btn')}</a>
          <button class="dashboard-toggle-btn" data-id="${l.id}" data-active="${l.is_active}">${l.is_active ? t('admin.hide_btn') : t('admin.restore_btn')}</button>
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
            showAdminBanner(d.message || t('dashboard.err_update_listing'));
            return;
          }
          loadListings();
        } catch (err) {
          showAdminBanner(t('common.error_connect'));
        }
      });
    });

    renderPager('listingsPager', data, (page) => {
      state.listings.page = page;
      loadListings();
    });
  } catch (err) {
    container.innerHTML = `<p class="dashboard-empty">${t('admin.err_loading_listings')}</p>`;
  }
}

async function loadOrders() {
  const container = document.getElementById('ordersTable');
  container.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const params = new URLSearchParams({ page: state.orders.page, page_size: 20 });
    if (state.orders.status) params.set('status', state.orders.status);
    const res = await authFetch(`/admin/orders?${params}`);
    const data = await res.json();
    if (!res.ok) {
      container.innerHTML = `<p class="dashboard-empty">${t('dashboard.err_fetch_orders')}</p>`;
      return;
    }
    if (data.items.length === 0) {
      container.innerHTML = `<p class="dashboard-empty">${t('admin.no_results')}</p>`;
      renderPager('ordersPager', data, () => {});
      return;
    }
    container.innerHTML = data.items
      .map(
        (o) => `
      <div class="dashboard-row" data-id="${o.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${t('dashboard.order_number', { id: o.id.slice(0, 8) })} · ${t('order_status.' + o.status)}</p>
          <p class="dashboard-row-meta">${t('admin.buyer_shop_meta', { buyer: escapeHtml(o.buyer_username || '—'), shop: escapeHtml(o.shop_name) })} · ${parseFloat(o.total_amount).toFixed(2)} ${o.currency} · ${new Date(o.created_at).toLocaleDateString('pl-PL')} · ${t('admin.payment_label', { status: t('payment_status_short.' + (o.payment_status === 'completed' ? 'completed' : o.payment_status === 'failed' ? 'failed' : 'pending')) })}</p>
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
    container.innerHTML = `<p class="dashboard-empty">${t('admin.err_loading_orders_admin')}</p>`;
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
    `<option value="">${t('admin.category_parent_none')}</option>` +
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
  document.getElementById('categorySubmitBtn').textContent = t('admin.add_category_submit');
}

function editCategory(cat) {
  document.getElementById('categoryId').value = cat.id;
  document.getElementById('categoryName').value = cat.name;
  document.getElementById('categoryDescription').value = cat.description || '';
  document.getElementById('categorySortOrder').value = cat.sort_order;
  populateCategoryParentSelect(cat.id);
  document.getElementById('categoryParent').value = cat.parent_id || '';
  document.getElementById('categorySubmitBtn').textContent = t('admin.save_category_submit');
  document.getElementById('categoryForm').style.display = 'flex';
}

async function loadCategories() {
  const container = document.getElementById('categoriesTable');
  container.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const res = await authFetch('/admin/categories');
    const items = await res.json();
    if (!res.ok) {
      container.innerHTML = `<p class="dashboard-empty">${t('admin.err_fetch_categories')}</p>`;
      return;
    }
    categoriesCache = items;
    const byId = Object.fromEntries(items.map((c) => [c.id, c]));

    if (items.length === 0) {
      container.innerHTML = `<p class="dashboard-empty">${t('admin.no_categories_yet')}</p>`;
      return;
    }

    container.innerHTML = items
      .map(
        (c) => `
      <div class="dashboard-row" data-id="${c.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(categoryPathName(c, byId))}</p>
          <p class="dashboard-row-meta">${t('admin.category_meta', { slug: escapeHtml(c.slug), order: c.sort_order })}${c.description ? ' · ' + escapeHtml(c.description) : ''}</p>
        </div>
        <div class="dashboard-row-actions">
          <button class="dashboard-edit-btn" data-id="${c.id}">${t('common.edit')}</button>
          <button class="dashboard-toggle-btn" data-id="${c.id}">${t('common.delete')}</button>
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
        if (!confirm(t('admin.confirm_delete_category', { name: cat.name }))) return;
        clearAdminBanner();
        try {
          const res = await authFetch(`/admin/categories/${cat.id}`, { method: 'DELETE' });
          const d = await res.json();
          if (!res.ok) {
            showAdminBanner(d.message || t('admin.err_delete_category'));
            return;
          }
          loadCategories();
        } catch (err) {
          showAdminBanner(t('common.error_connect'));
        }
      });
    });
  } catch (err) {
    container.innerHTML = `<p class="dashboard-empty">${t('admin.err_loading_categories')}</p>`;
  }
}

async function loadAuditLog() {
  const container = document.getElementById('auditTable');
  container.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const params = new URLSearchParams({ page: state.audit.page, page_size: 30 });
    const res = await authFetch(`/admin/audit-log?${params}`);
    const data = await res.json();
    if (!res.ok) {
      container.innerHTML = `<p class="dashboard-empty">${t('admin.err_fetch_audit')}</p>`;
      return;
    }
    if (data.items.length === 0) {
      container.innerHTML = `<p class="dashboard-empty">${t('admin.no_audit_entries')}</p>`;
      renderPager('auditPager', data, () => {});
      return;
    }
    container.innerHTML = data.items
      .map(
        (e) => `
      <div class="dashboard-row" data-id="${e.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${auditActionKeys[e.action] ? t(auditActionKeys[e.action]) : e.action}${e.details ? ': ' + escapeHtml(e.details) : ''}</p>
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
    container.innerHTML = `<p class="dashboard-empty">${t('admin.err_loading_audit')}</p>`;
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
        showAdminBanner(data.message || t('admin.err_save_category'));
        return;
      }
      document.getElementById('categoryForm').style.display = 'none';
      resetCategoryForm();
      loadCategories();
    } catch (err) {
      showAdminBanner(t('common.error_connect'));
    }
  });
}

function onPageLocaleChange() {
  const activeTab = document.querySelector('.dashboard-tab.active');
  if (activeTab) switchTab(activeTab.dataset.tab);
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
      showAdminBanner(t('admin.err_loading_platform'));
      return;
    }
    document.getElementById('adminMain').style.display = 'flex';
    const s = await res.json();
    renderOverviewFromStats(s);
  } catch (err) {
    showAdminBanner(t('common.error_connect'));
  }
}

init();
