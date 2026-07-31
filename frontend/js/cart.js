const CART_API_URL = window.API_URL || 'http://localhost:8080';
const CART_STORAGE_KEY = 'cart';

function cartKey(listingId, variantSkuId) {
  return variantSkuId ? `${listingId}__${variantSkuId}` : listingId;
}

function getCart() {
  try {
    return JSON.parse(localStorage.getItem(CART_STORAGE_KEY)) || {};
  } catch (err) {
    return {};
  }
}

function saveCart(cart) {
  localStorage.setItem(CART_STORAGE_KEY, JSON.stringify(cart));
  updateCartBadge();
}

// variantSkuId is null for listings without variants.
function addToCart(listingId, quantity = 1, variantSkuId = null) {
  const cart = getCart();
  const key = cartKey(listingId, variantSkuId);
  const existing = cart[key];
  cart[key] = {
    listingId,
    variantSkuId,
    quantity: (existing ? existing.quantity : 0) + quantity,
  };
  saveCart(cart);
}

function setCartQuantity(key, quantity) {
  const cart = getCart();
  if (quantity <= 0) {
    delete cart[key];
  } else if (cart[key]) {
    cart[key].quantity = quantity;
  }
  saveCart(cart);
  renderCartDrawer();
}

function removeFromCart(key) {
  const cart = getCart();
  delete cart[key];
  saveCart(cart);
  renderCartDrawer();
}

function getCartCount() {
  const cart = getCart();
  return Object.values(cart).reduce((sum, entry) => sum + entry.quantity, 0);
}

function updateCartBadge() {
  const badge = document.getElementById('cartCount');
  if (badge) badge.textContent = getCartCount();
}

function cartEscapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// Resolves the effective price/quantity/label for a cart entry against the
// full listing payload (which embeds variant_skus when has_variants is true).
function resolveCartLine(listing, entry) {
  if (!entry.variantSkuId) {
    return { price: parseFloat(listing.price), quantity: listing.quantity, label: null };
  }
  const sku = (listing.variant_skus || []).find((s) => s.id === entry.variantSkuId);
  if (!sku) return null;
  return {
    price: sku.price !== undefined && sku.price !== null ? parseFloat(sku.price) : parseFloat(listing.price),
    quantity: sku.quantity,
    label: sku.label,
  };
}

async function renderCartDrawer() {
  const listEl = document.getElementById('cartItemsList');
  const totalEl = document.getElementById('cartTotalAmount');
  if (!listEl) return;

  const cart = getCart();
  const entries = Object.entries(cart);

  if (entries.length === 0) {
    listEl.innerHTML = '<p class="cart-empty">Koszyk jest pusty</p>';
    if (totalEl) totalEl.textContent = '0.00 PLN';
    return;
  }

  listEl.innerHTML = '<p class="cart-loading">Ładowanie...</p>';

  const uniqueListingIds = [...new Set(entries.map(([, e]) => e.listingId))];
  const listingResults = await Promise.all(
    uniqueListingIds.map((id) =>
      fetch(`${CART_API_URL}/listings/${id}`)
        .then((res) => (res.ok ? res.json() : null))
        .catch(() => null)
    )
  );
  const listingsById = Object.fromEntries(uniqueListingIds.map((id, i) => [id, listingResults[i]]));

  let total = 0;
  listEl.innerHTML = '';

  entries.forEach(([key, entry]) => {
    const listing = listingsById[entry.listingId];
    if (!listing) return;
    const line = resolveCartLine(listing, entry);
    if (!line) return;

    const lineTotal = line.price * entry.quantity;
    total += lineTotal;

    const img = listing.primary_photo || 'https://picsum.photos/seed/placeholder/200/200';

    const row = document.createElement('div');
    row.className = 'cart-item-row';
    row.innerHTML = `
      <img src="${img}" alt="${cartEscapeHtml(listing.title)}" />
      <div class="cart-item-details">
        <p class="cart-item-title">${cartEscapeHtml(listing.title)}</p>
        ${line.label ? `<p class="cart-item-variant">${cartEscapeHtml(line.label)}</p>` : ''}
        <p class="cart-item-price">${line.price.toFixed(2)} ${listing.currency}</p>
        <div class="cart-item-qty-controls">
          <button data-action="dec">−</button>
          <span>${entry.quantity}</span>
          <button data-action="inc">+</button>
          <button data-action="remove" class="cart-item-remove">Usuń</button>
        </div>
      </div>
    `;

    row.querySelector('[data-action="dec"]').addEventListener('click', () => setCartQuantity(key, entry.quantity - 1));
    row.querySelector('[data-action="inc"]').addEventListener('click', () => {
      if (entry.quantity + 1 > line.quantity) return;
      setCartQuantity(key, entry.quantity + 1);
    });
    row.querySelector('[data-action="remove"]').addEventListener('click', () => removeFromCart(key));

    listEl.appendChild(row);
  });

  if (totalEl) totalEl.textContent = `${total.toFixed(2)} PLN`;
}

function toggleCartDrawer(forceOpen) {
  const drawer = document.getElementById('cartDrawer');
  const overlay = document.getElementById('cartOverlay');
  if (!drawer || !overlay) return;

  const shouldOpen = forceOpen !== undefined ? forceOpen : !drawer.classList.contains('open');
  drawer.classList.toggle('open', shouldOpen);
  overlay.classList.toggle('open', shouldOpen);

  if (shouldOpen) renderCartDrawer();
}

function initCartWidget() {
  updateCartBadge();

  const toggleBtn = document.getElementById('cartToggleBtn');
  const closeBtn = document.getElementById('closeCartBtn');
  const overlay = document.getElementById('cartOverlay');

  if (toggleBtn) toggleBtn.addEventListener('click', () => toggleCartDrawer());
  if (closeBtn) closeBtn.addEventListener('click', () => toggleCartDrawer(false));
  if (overlay) overlay.addEventListener('click', () => toggleCartDrawer(false));
}

document.addEventListener('DOMContentLoaded', initCartWidget);
