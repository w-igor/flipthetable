let categories = [];
let shippingProfiles = [];

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
      `<option value="">${t('dashboard.no_category')}</option>` +
      categories.map((c) => `<option value="${c.id}">${escapeHtml(c.name)}</option>`).join('');

    const filterSelect = document.getElementById('listingFilterCategory');
    if (filterSelect) {
      filterSelect.innerHTML =
        `<option value="">${t('dashboard.filter_all_categories')}</option>` +
        categories.map((c) => `<option value="${c.id}">${escapeHtml(c.name)}</option>`).join('');
    }
  } catch (err) {
    console.error('Nie udało się pobrać kategorii', err);
  }
}

async function loadShippingProfilesForSelect() {
  try {
    const res = await authFetch('/seller/shipping-profiles');
    if (!res.ok) return;
    shippingProfiles = await res.json();
    const select = document.getElementById('listingShippingProfile');
    select.innerHTML =
      `<option value="">${t('dashboard.no_shipping_profile')}</option>` +
      shippingProfiles
        .map((sp) => `<option value="${sp.id}">${escapeHtml(sp.name)} — ${parseFloat(sp.price).toFixed(2)} PLN, ${sp.min_days === sp.max_days ? t('common.days_single', { n: sp.min_days }) : t('common.days_range', { min: sp.min_days, max: sp.max_days })}</option>`)
        .join('');
  } catch (err) {
    console.error('Nie udało się pobrać profili wysyłki', err);
  }
}

function resetShippingProfileForm() {
  document.getElementById('shippingProfileId').value = '';
  document.getElementById('shippingProfileName').value = '';
  document.getElementById('shippingProfilePrice').value = '';
  document.getElementById('shippingProfileMinDays').value = '';
  document.getElementById('shippingProfileMaxDays').value = '';
  document.getElementById('shippingProfileSubmitBtn').textContent = t('dashboard.add_profile_submit');
}

function editShippingProfile(sp) {
  document.getElementById('shippingProfileId').value = sp.id;
  document.getElementById('shippingProfileName').value = sp.name;
  document.getElementById('shippingProfilePrice').value = sp.price;
  document.getElementById('shippingProfileMinDays').value = sp.min_days;
  document.getElementById('shippingProfileMaxDays').value = sp.max_days;
  document.getElementById('shippingProfileSubmitBtn').textContent = t('common.save_changes');
  document.getElementById('shippingProfileForm').style.display = 'flex';
}

async function loadShippingProfiles() {
  const container = document.getElementById('shippingProfilesTable');
  container.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const res = await authFetch('/seller/shipping-profiles');
    const items = await res.json();
    if (!res.ok) {
      container.innerHTML = `<p class="dashboard-empty">${t('dashboard.err_fetch_shipping_profiles')}</p>`;
      return;
    }
    shippingProfiles = items;
    if (items.length === 0) {
      container.innerHTML = `<p class="dashboard-empty">${t('dashboard.no_shipping_profiles_yet')}</p>`;
      return;
    }
    container.innerHTML = items
      .map(
        (sp) => `
      <div class="dashboard-row" data-id="${sp.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(sp.name)}</p>
          <p class="dashboard-row-meta">${parseFloat(sp.price).toFixed(2)} PLN · ${sp.min_days === sp.max_days ? t('common.days_single', { n: sp.min_days }) : t('common.days_range', { min: sp.min_days, max: sp.max_days })} ${t('shipping_profile.delivery_suffix')}</p>
        </div>
        <div class="dashboard-row-actions">
          <button class="dashboard-edit-btn" data-id="${sp.id}">${t('common.edit')}</button>
          <button class="dashboard-toggle-btn" data-id="${sp.id}">${t('common.delete')}</button>
        </div>
      </div>
    `
      )
      .join('');

    container.querySelectorAll('.dashboard-edit-btn').forEach((btn) => {
      btn.addEventListener('click', () => {
        const sp = items.find((s) => s.id === btn.dataset.id);
        if (sp) editShippingProfile(sp);
      });
    });
    container.querySelectorAll('.dashboard-toggle-btn').forEach((btn) => {
      btn.addEventListener('click', async () => {
        const sp = items.find((s) => s.id === btn.dataset.id);
        if (!sp) return;
        if (!confirm(t('dashboard.confirm_delete_shipping_profile', { name: sp.name }))) return;
        try {
          const res = await authFetch(`/seller/shipping-profiles/${sp.id}`, { method: 'DELETE' });
          const d = await res.json();
          if (!res.ok) {
            showDashboardBanner(d.message || t('dashboard.err_delete_shipping_profile'));
            return;
          }
          loadShippingProfiles();
          loadShippingProfilesForSelect();
        } catch (err) {
          showDashboardBanner(t('common.error_connect'));
        }
      });
    });
  } catch (err) {
    container.innerHTML = `<p class="dashboard-empty">${t('dashboard.err_loading_shipping_profiles')}</p>`;
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
      showDashboardBanner(t('dashboard.err_upload_photo'));
      return;
    }
    document.getElementById(hiddenInputId).value = url;
    setPhotoPreview(previewId, url);
  });
}

const MAX_LISTING_PHOTOS = 8;
let listingPhotoUrls = [];

function renderListingPhotosGrid() {
  const grid = document.getElementById('listingPhotosGrid');
  grid.innerHTML = listingPhotoUrls
    .map(
      (url, i) => `
    <div class="dashboard-photo-thumb ${i === 0 ? 'is-primary' : ''}" data-index="${i}">
      <img src="${url}" alt="${t('dashboard.photo_alt', { n: i + 1 })}" />
      <button type="button" class="dashboard-photo-thumb-remove" data-index="${i}" title="${t('dashboard.remove_photo_title')}">&times;</button>
    </div>
  `
    )
    .join('');

  grid.querySelectorAll('.dashboard-photo-thumb-remove').forEach((btn) => {
    btn.addEventListener('click', () => {
      listingPhotoUrls.splice(parseInt(btn.dataset.index, 10), 1);
      renderListingPhotosGrid();
    });
  });

  document.getElementById('listingPhotoFile').disabled = listingPhotoUrls.length >= MAX_LISTING_PHOTOS;
}

function bindListingPhotosUpload() {
  document.getElementById('listingPhotoFile').addEventListener('change', async (e) => {
    const files = Array.from(e.target.files || []);
    e.target.value = '';
    if (files.length === 0) return;
    clearDashboardBanner();

    const remaining = MAX_LISTING_PHOTOS - listingPhotoUrls.length;
    if (files.length > remaining) {
      showDashboardBanner(t('dashboard.photos_remaining_limit', { remaining, max: MAX_LISTING_PHOTOS }));
    }

    for (const file of files.slice(0, remaining)) {
      const url = await uploadPhoto(file);
      if (!url) {
        showDashboardBanner(t('dashboard.err_upload_one_photo'));
        continue;
      }
      listingPhotoUrls.push(url);
    }
    renderListingPhotosGrid();
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
    showDashboardBanner(t('dashboard.err_fetch_shop'));
    return null;
  }
  const shop = await res.json();
  document.getElementById('noShopState').style.display = 'none';
  document.getElementById('dashboardMain').style.display = 'flex';
  fillEditShopForm(shop);
  loadStats();
  return shop;
}

async function loadStats() {
  const grid = document.getElementById('statsGrid');
  grid.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const res = await authFetch('/seller/stats');
    const stats = await res.json();
    if (!res.ok) {
      grid.innerHTML = `<p class="dashboard-empty">${t('dashboard.err_fetch_stats')}</p>`;
      return;
    }
    grid.innerHTML = `
      <div class="dashboard-stat-tile">
        <p class="dashboard-stat-value">${stats.total_orders}</p>
        <p class="dashboard-stat-label">${t('dashboard.stat_orders')}</p>
      </div>
      <div class="dashboard-stat-tile">
        <p class="dashboard-stat-value">${parseFloat(stats.total_revenue).toFixed(2)} PLN</p>
        <p class="dashboard-stat-label">${t('dashboard.stat_revenue')}</p>
      </div>
      <div class="dashboard-stat-tile">
        <p class="dashboard-stat-value">${stats.listings_active}/${stats.listings_total}</p>
        <p class="dashboard-stat-label">${t('dashboard.stat_active_listings')}</p>
      </div>
    `;
  } catch (err) {
    grid.innerHTML = `<p class="dashboard-empty">${t('dashboard.err_loading_stats')}</p>`;
  }
}

function switchTab(tabName) {
  document.querySelectorAll('.dashboard-tab').forEach((b) => b.classList.toggle('active', b.dataset.tab === tabName));
  document.querySelectorAll('.dashboard-tab-panel').forEach((p) => {
    p.style.display = p.id === `tab-${tabName}` ? 'block' : 'none';
  });
  if (tabName === 'listings') loadListings();
  if (tabName === 'orders') loadOrders();
  if (tabName === 'shipping') loadShippingProfiles();
}

function resetListingForm() {
  document.getElementById('listingId').value = '';
  document.getElementById('listingTitle').value = '';
  document.getElementById('listingDescription').value = '';
  document.getElementById('listingPrice').value = '';
  document.getElementById('listingQuantity').value = '1';
  document.getElementById('listingCategory').value = '';
  document.getElementById('listingPhotoFile').value = '';
  document.getElementById('listingShippingProfile').value = '';
  listingPhotoUrls = [];
  renderListingPhotosGrid();
  document.getElementById('listingSubmitBtn').textContent = t('dashboard.add_listing_submit');
  document.getElementById('variantQuantityHint').style.display = 'none';
  resetVariantEditor();
}

async function editListing(item) {
  document.getElementById('listingId').value = item.id;
  document.getElementById('listingTitle').value = item.title;
  document.getElementById('listingDescription').value = item.description || '';
  document.getElementById('listingPrice').value = item.price;
  document.getElementById('listingQuantity').value = item.quantity;
  document.getElementById('listingCategory').value = item.category_id || '';
  document.getElementById('listingShippingProfile').value = item.shipping_profile_id || '';
  document.getElementById('listingPhotoFile').value = '';
  listingPhotoUrls = item.primary_photo ? [item.primary_photo] : [];
  renderListingPhotosGrid();
  document.getElementById('listingSubmitBtn').textContent = t('common.save_changes');
  document.getElementById('variantQuantityHint').style.display = item.has_variants ? 'block' : 'none';
  document.getElementById('listingForm').style.display = 'flex';

  try {
    const res = await authFetch(`/seller/listings/${item.id}`);
    if (res.ok) {
      const full = await res.json();
      listingPhotoUrls = (full.photos || []).map((p) => p.url);
      renderListingPhotosGrid();
      if (item.has_variants) populateVariantEditor(full);
    }
  } catch (err) {
    if (item.has_variants) showVariantBanner(t('dashboard.err_fetch_variants'));
  }
  if (!item.has_variants) resetVariantEditor();
}

async function toggleListingActive(item) {
  clearDashboardBanner();
  try {
    const res = await authFetch(`/listings/${item.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: item.title,
        description: item.description || '',
        category_id: item.category_id || null,
        price: item.price,
        currency: item.currency,
        quantity: item.quantity,
        shipping_profile_id: item.shipping_profile_id || null,
        is_active: !item.is_active,
      }),
    });
    const data = await res.json();
    if (!res.ok) {
      showDashboardBanner(data.message || t('dashboard.err_update_listing'));
      return;
    }
    loadListings();
  } catch (err) {
    showDashboardBanner(t('common.error_connect'));
  }
}

async function deleteListing(item) {
  clearDashboardBanner();
  if (!confirm(t('dashboard.confirm_delete_listing', { title: item.title }))) return;
  try {
    const res = await authFetch(`/listings/${item.id}`, { method: 'DELETE' });
    const data = await res.json();
    if (!res.ok) {
      showDashboardBanner(data.message || t('dashboard.err_delete_listing'));
      return;
    }
    loadListings();
  } catch (err) {
    showDashboardBanner(t('common.error_connect'));
  }
}

function buildListingFilterQuery() {
  const params = new URLSearchParams();
  const search = document.getElementById('listingFilterSearch').value.trim();
  const status = document.getElementById('listingFilterStatus').value;
  const categoryId = document.getElementById('listingFilterCategory').value;
  const sort = document.getElementById('listingFilterSort').value;
  if (search) params.set('q', search);
  if (status) params.set('status', status);
  if (categoryId) params.set('category_id', categoryId);
  if (sort) params.set('sort', sort);
  const qs = params.toString();
  return qs ? `?${qs}` : '';
}

async function loadListings() {
  const container = document.getElementById('listingsTable');
  container.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const filterQuery = buildListingFilterQuery();
    const res = await authFetch(`/seller/listings${filterQuery}`);
    const items = await res.json();
    if (!res.ok) {
      container.innerHTML = `<p class="dashboard-empty">${t('dashboard.err_fetch_listings')}</p>`;
      return;
    }
    if (items.length === 0) {
      container.innerHTML = `<p class="dashboard-empty">${filterQuery ? t('dashboard.no_filtered_listings') : t('dashboard.no_listings_yet')}</p>`;
      return;
    }

    container.innerHTML = items
      .map(
        (item) => `
      <div class="dashboard-row" data-id="${item.id}">
        <img src="${item.primary_photo || 'https://picsum.photos/seed/placeholder/100/100'}" alt="${escapeHtml(item.title)}" />
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${escapeHtml(item.title)}</p>
          <p class="dashboard-row-meta">${parseFloat(item.price).toFixed(2)} ${item.currency} · ${item.quantity} ${t('dashboard.units_suffix')} · ${item.is_active ? t('dashboard.status_active_f') : t('dashboard.status_inactive_f')}</p>
        </div>
        <div class="dashboard-row-actions">
          <button class="dashboard-edit-btn" data-id="${item.id}">${t('common.edit')}${item.has_variants ? t('dashboard.variants_suffix') : ''}</button>
          <button class="dashboard-toggle-btn" data-id="${item.id}">${item.is_active ? t('dashboard.toggle_disable') : t('dashboard.toggle_enable')}</button>
          <button class="dashboard-delete-btn" data-id="${item.id}">${t('common.delete')}</button>
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
    container.querySelectorAll('.dashboard-delete-btn').forEach((btn) => {
      btn.addEventListener('click', () => {
        const item = items.find((i) => i.id === btn.dataset.id);
        if (item) deleteListing(item);
      });
    });
  } catch (err) {
    container.innerHTML = `<p class="dashboard-empty">${t('dashboard.err_loading_listings')}</p>`;
  }
}

let variantCombos = []; // [{ values: [v1, v2?], price: string, quantity: number }]

function showVariantBanner(message, type = 'error') {
  const banner = document.getElementById('variantEditorBanner');
  banner.textContent = message;
  banner.className = `dashboard-banner show ${type}`;
}

function clearVariantBanner() {
  const banner = document.getElementById('variantEditorBanner');
  banner.className = 'dashboard-banner';
  banner.textContent = '';
}

function showEtsyBanner(message, type = 'error') {
  const banner = document.getElementById('etsyImportBanner');
  banner.innerHTML = message;
  banner.className = `dashboard-banner show ${type}`;
}

function clearEtsyBanner() {
  const banner = document.getElementById('etsyImportBanner');
  banner.className = 'dashboard-banner';
  banner.textContent = '';
}

async function loadEtsyStatus() {
  try {
    const res = await authFetch('/seller/etsy/status');
    if (!res.ok) return;
    const status = await res.json();

    document.getElementById('etsyNotConfiguredHint').style.display = status.api_key_configured ? 'none' : 'block';
    const canUseEtsy = status.api_key_configured;
    document.getElementById('etsyNotConnected').style.display = canUseEtsy && !status.connected ? 'block' : 'none';
    document.getElementById('etsyImportFields').style.display = canUseEtsy && status.connected ? 'block' : 'none';
    document.getElementById('etsyImportSubmitBtn').style.display = canUseEtsy && status.connected ? '' : 'none';
    if (status.connected && status.etsy_shop_id) {
      document.getElementById('etsyConnectedShopId').textContent = status.etsy_shop_id;
    }
  } catch (err) {
    console.error('Nie udało się pobrać statusu Etsy', err);
  }
}

function etsyImportResultMessage(data) {
  const parts = [t('dashboard.etsy_result_imported', { n: data.imported })];
  if (data.skipped) parts.push(t('dashboard.etsy_result_skipped', { n: data.skipped }));
  if (data.errors && data.errors.length) parts.push(t('dashboard.etsy_result_errors', { n: data.errors.length }));
  let html = escapeHtml(parts.join(' · '));
  if (data.errors && data.errors.length) {
    html +=
      '<ul style="margin:8px 0 0;padding-left:18px;">' +
      data.errors
        .map((e) => `<li>#${escapeHtml(e.etsy_listing_id)}: ${escapeHtml(e.message)}</li>`)
        .join('') +
      '</ul>';
  }
  return html;
}

function addVariantTypeRow(name = '', optionsStr = '') {
  const container = document.getElementById('variantTypeRows');
  if (container.children.length >= 2) return;
  const row = document.createElement('div');
  row.className = 'variant-type-row';
  row.innerHTML = `
    <div class="dashboard-field">
      <label>${t('dashboard.variant_type_name_label')}</label>
      <input type="text" class="variant-type-name" value="${escapeHtml(name)}" />
    </div>
    <div class="dashboard-field">
      <label>${t('dashboard.variant_type_values_label')}</label>
      <input type="text" class="variant-type-options" value="${escapeHtml(optionsStr)}" placeholder="${t('dashboard.variant_type_values_placeholder')}" />
    </div>
    <button type="button" class="dashboard-cancel-btn dashboard-btn-small remove-variant-type-btn">${t('dashboard.remove_variant_type_btn')}</button>
  `;
  row.querySelector('.remove-variant-type-btn').addEventListener('click', () => row.remove());
  container.appendChild(row);
}

function readVariantTypesFromInputs() {
  const rows = document.querySelectorAll('#variantTypeRows .variant-type-row');
  const types = [];
  rows.forEach((row) => {
    const name = row.querySelector('.variant-type-name').value.trim();
    const options = row
      .querySelector('.variant-type-options')
      .value.split(',')
      .map((v) => v.trim())
      .filter((v) => v);
    if (name && options.length > 0) {
      types.push({ name, options });
    }
  });
  return types;
}

function comboKey(values) {
  return values.join('|||');
}

function pricesVaryEnabled() {
  return document.getElementById('variantPricesVary').checked;
}

function readCombosFromTable() {
  const pricesVary = pricesVaryEnabled();
  document.querySelectorAll('#variantCombosWrap tbody tr').forEach((tr) => {
    const key = tr.dataset.key;
    const combo = variantCombos.find((c) => comboKey(c.values) === key);
    if (!combo) return;
    const priceInput = tr.querySelector('.variant-combo-price');
    combo.price = pricesVary && priceInput ? priceInput.value.trim() : '';
    combo.quantity = parseInt(tr.querySelector('.variant-combo-qty').value, 10) || 0;
  });
}

function renderComboTable(types) {
  const wrap = document.getElementById('variantCombosWrap');
  if (variantCombos.length === 0) {
    wrap.innerHTML = `<p class="dashboard-empty">${t('dashboard.combos_empty_hint')}</p>`;
    return;
  }
  const pricesVary = pricesVaryEnabled();
  const headerLabels = types.map((vt) => escapeHtml(vt.name));
  wrap.innerHTML = `
    <table class="variant-combo-table">
      <thead>
        <tr>
          ${headerLabels.map((h) => `<th>${h}</th>`).join('')}
          ${pricesVary ? `<th>${t('dashboard.profile_price_label')}</th>` : ''}
          <th>${t('dashboard.quantity_label')}</th>
        </tr>
      </thead>
      <tbody>
        ${variantCombos
          .map(
            (c) => `
          <tr data-key="${escapeHtml(comboKey(c.values))}">
            ${c.values.map((v) => `<td>${escapeHtml(v)}</td>`).join('')}
            ${pricesVary ? `<td><input type="number" step="0.01" min="0" class="variant-combo-price" value="${c.price || ''}" placeholder="${t('dashboard.combo_price_placeholder')}" /></td>` : ''}
            <td><input type="number" step="1" min="0" class="variant-combo-qty" value="${c.quantity}" /></td>
          </tr>
        `
          )
          .join('')}
      </tbody>
    </table>
  `;
}

function generateCombos() {
  readCombosFromTable();
  const types = readVariantTypesFromInputs();
  if (types.length === 0) {
    showVariantBanner(t('dashboard.err_variant_type_required'));
    return;
  }
  clearVariantBanner();

  let combinations = [[]];
  types.forEach((t) => {
    const next = [];
    combinations.forEach((prefix) => {
      t.options.forEach((opt) => next.push([...prefix, opt]));
    });
    combinations = next;
  });

  const existingByKey = Object.fromEntries(variantCombos.map((c) => [comboKey(c.values), c]));
  variantCombos = combinations.map((values) => {
    const existing = existingByKey[comboKey(values)];
    return existing || { values, price: '', quantity: 0 };
  });

  renderComboTable(types);
}

let editingItemHadVariants = false;

// Resets the inline variant editor to its empty, collapsed state (new listing / no variants).
function resetVariantEditor() {
  clearVariantBanner();
  editingItemHadVariants = false;
  document.getElementById('listingHasVariants').checked = false;
  document.getElementById('variantPricesVary').checked = false;
  document.getElementById('variantEditor').style.display = 'none';
  document.getElementById('variantTypeRows').innerHTML = '';
  variantCombos = [];
  document.getElementById('variantCombosWrap').innerHTML = '';
}

// Fills the inline variant editor from an existing listing's full detail (with variant_types/variant_skus).
function populateVariantEditor(full) {
  clearVariantBanner();
  editingItemHadVariants = !!full.has_variants;
  document.getElementById('listingHasVariants').checked = editingItemHadVariants;
  document.getElementById('variantTypeRows').innerHTML = '';
  variantCombos = [];
  document.getElementById('variantCombosWrap').innerHTML = '';

  const types = full.variant_types || [];
  if (!editingItemHadVariants || types.length === 0) {
    document.getElementById('variantEditor').style.display = 'none';
    return;
  }

  types.forEach((t) => addVariantTypeRow(t.name, t.options.map((o) => o.value).join(', ')));
  const optionValueById = {};
  types.forEach((t) => t.options.forEach((o) => { optionValueById[o.id] = o.value; }));
  variantCombos = (full.variant_skus || []).map((sku) => {
    const values = [sku.option1_id, sku.option2_id].filter(Boolean).map((id) => optionValueById[id]);
    return { values, price: sku.price || '', quantity: sku.quantity };
  });
  document.getElementById('variantPricesVary').checked = variantCombos.some((c) => c.price);
  renderComboTable(types);
  document.getElementById('variantEditor').style.display = 'block';
}

// Reads the current variant editor state into the {types, skus} shape the API expects,
// or null (with a banner message) if the checkbox is on but the config is incomplete.
function buildVariantsPayload() {
  const enabled = document.getElementById('listingHasVariants').checked;
  if (!enabled) {
    return { types: [], skus: [] };
  }
  readCombosFromTable();
  const types = readVariantTypesFromInputs();
  if (types.length === 0) {
    showVariantBanner(t('dashboard.err_variant_type_required'));
    return null;
  }
  if (variantCombos.length === 0) {
    showVariantBanner(t('dashboard.err_generate_combos_first'));
    return null;
  }
  return {
    types: types.map((t) => ({ name: t.name, options: t.options })),
    skus: variantCombos.map((c) => ({
      option_values: c.values,
      price: c.price ? c.price : null,
      quantity: c.quantity,
    })),
  };
}

// Persists the variant configuration for a just-created/just-updated listing. Only called
// when it's actually needed: the checkbox is checked, or variants existed before and were removed.
async function saveVariantsForListing(listingId, payload) {
  const res = await authFetch(`/listings/${listingId}/variants`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  const data = await res.json();
  if (!res.ok) {
    throw new Error(data.message || t('dashboard.err_save_variants'));
  }
}

async function loadOrders() {
  const container = document.getElementById('ordersTable');
  container.innerHTML = `<p class="dashboard-empty">${t('common.loading')}</p>`;
  try {
    const res = await authFetch('/seller/orders');
    const orders = await res.json();
    if (!res.ok) {
      container.innerHTML = `<p class="dashboard-empty">${t('dashboard.err_fetch_orders')}</p>`;
      return;
    }
    if (orders.length === 0) {
      container.innerHTML = `<p class="dashboard-empty">${t('dashboard.no_orders')}</p>`;
      return;
    }

    const orderStatusValues = ['pending', 'paid', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded'];
    container.innerHTML = orders
      .map(
        (o) => `
      <div class="dashboard-row" data-id="${o.id}">
        <div class="dashboard-row-info">
          <p class="dashboard-row-title">${t('dashboard.order_number', { id: o.id.slice(0, 8) })}</p>
          <p class="dashboard-row-meta">${t('dashboard.buyer_label', { name: escapeHtml(o.buyer_username || '—') })} · ${parseFloat(o.total_amount).toFixed(2)} ${o.currency}${parseFloat(o.shipping_amount) > 0 ? t('dashboard.shipping_included_paren', { amount: parseFloat(o.shipping_amount).toFixed(2) + ' ' + o.currency }) : ''} · ${new Date(o.created_at).toLocaleDateString('pl-PL')} · ${t('dashboard.payment_label', { status: t('payment_status_short.' + (o.payment_status === 'completed' ? 'completed' : o.payment_status === 'failed' ? 'failed' : 'pending')) })}</p>
        </div>
        <select class="dashboard-order-status" data-id="${o.id}">
          ${orderStatusValues
            .map((value) => `<option value="${value}" ${value === o.status ? 'selected' : ''}>${t('order_status.' + value)}</option>`)
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
            showDashboardBanner(data.message || t('dashboard.err_update_status'));
          }
        } catch (err) {
          showDashboardBanner(t('common.error_connect'));
        }
      });
    });
  } catch (err) {
    container.innerHTML = `<p class="dashboard-empty">${t('dashboard.err_loading_orders')}</p>`;
  }
}

function bindEvents() {
  bindPhotoUpload('shopAvatarFile', 'shopAvatarUrl', 'shopAvatarPreview');
  bindPhotoUpload('shopBannerFile', 'shopBannerUrl', 'shopBannerPreview');
  bindPhotoUpload('editShopAvatarFile', 'editShopAvatarUrl', 'editShopAvatarPreview');
  bindPhotoUpload('editShopBannerFile', 'editShopBannerUrl', 'editShopBannerPreview');
  bindListingPhotosUpload();

  let listingFilterDebounce = null;
  document.getElementById('listingFilterSearch').addEventListener('input', () => {
    clearTimeout(listingFilterDebounce);
    listingFilterDebounce = setTimeout(loadListings, 300);
  });
  document.getElementById('listingFilterStatus').addEventListener('change', loadListings);
  document.getElementById('listingFilterCategory').addEventListener('change', loadListings);
  document.getElementById('listingFilterSort').addEventListener('change', loadListings);

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
        showDashboardBanner(data.message || t('dashboard.err_create_shop'));
        return;
      }

      const storage = localStorage.getItem('user') ? localStorage : sessionStorage;
      const user = JSON.parse(storage.getItem('user'));
      user.is_seller = true;
      storage.setItem('user', JSON.stringify(user));
      updateHeaderForAuth();

      document.getElementById('noShopState').style.display = 'none';
      document.getElementById('dashboardMain').style.display = 'flex';
      fillEditShopForm(data);
      switchTab('shop');
    } catch (err) {
      showDashboardBanner(t('common.error_connect'));
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
        showDashboardBanner(data.message || t('dashboard.err_save_shop'));
        return;
      }
      showDashboardBanner(t('dashboard.saved_changes'), 'success');
    } catch (err) {
      showDashboardBanner(t('common.error_connect'));
    }
  });

  document.querySelectorAll('.dashboard-tab').forEach((btn) => {
    btn.addEventListener('click', () => switchTab(btn.dataset.tab));
  });

  document.getElementById('addVariantTypeBtn').addEventListener('click', () => addVariantTypeRow());
  document.getElementById('generateCombosBtn').addEventListener('click', generateCombos);
  document.getElementById('variantPricesVary').addEventListener('change', () => {
    readCombosFromTable();
    renderComboTable(readVariantTypesFromInputs());
  });

  document.getElementById('newShippingProfileBtn').addEventListener('click', () => {
    resetShippingProfileForm();
    document.getElementById('shippingProfileForm').style.display = 'flex';
  });

  document.getElementById('cancelShippingProfileBtn').addEventListener('click', () => {
    document.getElementById('shippingProfileForm').style.display = 'none';
    resetShippingProfileForm();
  });

  document.getElementById('shippingProfileForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    clearDashboardBanner();
    const id = document.getElementById('shippingProfileId').value;
    const payload = {
      name: document.getElementById('shippingProfileName').value.trim(),
      price: document.getElementById('shippingProfilePrice').value,
      min_days: parseInt(document.getElementById('shippingProfileMinDays').value, 10) || 0,
      max_days: parseInt(document.getElementById('shippingProfileMaxDays').value, 10) || 0,
    };
    try {
      const res = await authFetch(id ? `/seller/shipping-profiles/${id}` : '/seller/shipping-profiles', {
        method: id ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (!res.ok) {
        showDashboardBanner(data.message || t('dashboard.err_save_shipping_profile'));
        return;
      }
      document.getElementById('shippingProfileForm').style.display = 'none';
      resetShippingProfileForm();
      loadShippingProfiles();
      loadShippingProfilesForSelect();
    } catch (err) {
      showDashboardBanner(t('common.error_connect'));
    }
  });

  document.getElementById('newListingBtn').addEventListener('click', () => {
    resetListingForm();
    document.getElementById('listingForm').style.display = 'flex';
  });

  document.getElementById('etsyImportBtn').addEventListener('click', () => {
    clearEtsyBanner();
    document.getElementById('listingForm').style.display = 'none';
    const panel = document.getElementById('etsyImportPanel');
    panel.style.display = panel.style.display === 'none' ? 'flex' : 'none';
    if (panel.style.display === 'flex') loadEtsyStatus();
  });

  document.getElementById('etsyImportCancelBtn').addEventListener('click', () => {
    document.getElementById('etsyImportPanel').style.display = 'none';
    clearEtsyBanner();
  });

  document.querySelectorAll('input[name="etsyImportMode"]').forEach((radio) => {
    radio.addEventListener('change', () => {
      document.getElementById('etsyListingIdsField').style.display =
        document.getElementById('etsyModeSelected').checked ? 'block' : 'none';
    });
  });

  document.getElementById('etsyConnectBtn').addEventListener('click', async () => {
    clearEtsyBanner();
    try {
      const res = await authFetch('/seller/etsy/oauth/start');
      const data = await res.json();
      if (!res.ok) {
        showEtsyBanner(escapeHtml(data.message || t('dashboard.etsy_err_import_failed')));
        return;
      }
      window.location.href = data.url;
    } catch (err) {
      showEtsyBanner(t('common.error_connect'));
    }
  });

  document.getElementById('etsyDisconnectBtn').addEventListener('click', async () => {
    clearEtsyBanner();
    try {
      const res = await authFetch('/seller/etsy/connection', { method: 'DELETE' });
      if (!res.ok) {
        showEtsyBanner(t('dashboard.etsy_err_import_failed'));
        return;
      }
      loadEtsyStatus();
    } catch (err) {
      showEtsyBanner(t('common.error_connect'));
    }
  });

  document.getElementById('etsyImportSubmitBtn').addEventListener('click', async () => {
    clearEtsyBanner();
    const mode = document.getElementById('etsyModeSelected').checked ? 'selected' : 'all';
    const listingIds = document
      .getElementById('etsyListingIdsInput')
      .value.split(/[,\s]+/)
      .map((s) => s.trim())
      .filter(Boolean);

    if (mode === 'selected' && listingIds.length === 0) {
      showEtsyBanner(t('dashboard.etsy_err_ids_required'));
      return;
    }

    const btn = document.getElementById('etsyImportSubmitBtn');
    btn.disabled = true;
    showEtsyBanner(t('dashboard.etsy_importing'), 'success');

    try {
      const res = await authFetch('/seller/etsy/import', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mode, listing_ids: listingIds }),
      });
      const data = await res.json();
      if (!res.ok) {
        showEtsyBanner(escapeHtml(data.message || t('dashboard.etsy_err_import_failed')));
        return;
      }
      showEtsyBanner(etsyImportResultMessage(data), data.imported > 0 ? 'success' : 'error');
      loadListings();
    } catch (err) {
      showEtsyBanner(t('common.error_connect'));
    } finally {
      btn.disabled = false;
    }
  });

  document.getElementById('cancelListingBtn').addEventListener('click', () => {
    document.getElementById('listingForm').style.display = 'none';
    resetListingForm();
  });

  document.getElementById('listingHasVariants').addEventListener('change', (e) => {
    document.getElementById('variantEditor').style.display = e.target.checked ? 'block' : 'none';
    if (e.target.checked && document.getElementById('variantTypeRows').children.length === 0) {
      addVariantTypeRow();
    }
  });

  document.getElementById('listingForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    clearDashboardBanner();
    const id = document.getElementById('listingId').value;

    const variantsPayload = buildVariantsPayload();
    if (variantsPayload === null) return;

    const payload = {
      title: document.getElementById('listingTitle').value.trim(),
      description: document.getElementById('listingDescription').value.trim(),
      category_id: document.getElementById('listingCategory').value || null,
      price: document.getElementById('listingPrice').value,
      currency: 'PLN',
      quantity: parseInt(document.getElementById('listingQuantity').value, 10) || 0,
      photos: listingPhotoUrls,
      shipping_profile_id: document.getElementById('listingShippingProfile').value || null,
    };

    try {
      const res = await authFetch(id ? `/listings/${id}` : '/listings', {
        method: id ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await res.json();
      if (!res.ok) {
        showDashboardBanner(data.message || t('dashboard.err_save_listing'));
        return;
      }

      const shouldSyncVariants = variantsPayload.types.length > 0 || editingItemHadVariants;
      if (shouldSyncVariants) {
        try {
          await saveVariantsForListing(data.id, variantsPayload);
        } catch (err) {
          showDashboardBanner(err.message || t('dashboard.listing_saved_variants_failed'));
          loadListings();
          return;
        }
      }

      document.getElementById('listingForm').style.display = 'none';
      resetListingForm();
      loadListings();
    } catch (err) {
      showDashboardBanner(t('common.error_connect'));
    }
  });
}

function onPageLocaleChange() {
  loadStats();
  loadListings();
  loadOrders();
  loadShippingProfiles();
}

function handleEtsyOAuthRedirectParams() {
  const params = new URLSearchParams(window.location.search);
  const connected = params.get('etsy_connected');
  const error = params.get('etsy_error');
  if (!connected && !error) return;

  switchTab('listings');
  document.getElementById('etsyImportPanel').style.display = 'flex';
  loadEtsyStatus();
  if (connected) {
    showEtsyBanner(t('dashboard.etsy_connected_success'), 'success');
  } else {
    showEtsyBanner(escapeHtml(error));
  }

  params.delete('etsy_connected');
  params.delete('etsy_error');
  const query = params.toString();
  window.history.replaceState({}, '', window.location.pathname + (query ? `?${query}` : ''));
}

async function init() {
  if (!requireLogin()) return;
  updateHeaderForAuth();
  bindEvents();
  await loadCategories();
  await loadShippingProfilesForSelect();
  await loadMyShop();
  handleEtsyOAuthRedirectParams();
}

init();
