const CART_API_URL = window.API_URL || 'http://localhost:8080';
const CART_STORAGE_KEY = 'cart';

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

function addToCart(listingId, quantity = 1) {
  const cart = getCart();
  cart[listingId] = (cart[listingId] || 0) + quantity;
  saveCart(cart);
}

function setCartQuantity(listingId, quantity) {
  const cart = getCart();
  if (quantity <= 0) {
    delete cart[listingId];
  } else {
    cart[listingId] = quantity;
  }
  saveCart(cart);
  renderCartDrawer();
}

function removeFromCart(listingId) {
  const cart = getCart();
  delete cart[listingId];
  saveCart(cart);
  renderCartDrawer();
}

function getCartCount() {
  const cart = getCart();
  return Object.values(cart).reduce((sum, qty) => sum + qty, 0);
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

async function renderCartDrawer() {
  const listEl = document.getElementById('cartItemsList');
  const totalEl = document.getElementById('cartTotalAmount');
  if (!listEl) return;

  const cart = getCart();
  const listingIds = Object.keys(cart);

  if (listingIds.length === 0) {
    listEl.innerHTML = '<p class="cart-empty">Koszyk jest pusty</p>';
    if (totalEl) totalEl.textContent = '0.00 PLN';
    return;
  }

  listEl.innerHTML = '<p class="cart-loading">Ładowanie...</p>';

  const results = await Promise.all(
    listingIds.map((id) =>
      fetch(`${CART_API_URL}/listings/${id}`)
        .then((res) => (res.ok ? res.json() : null))
        .catch(() => null)
    )
  );

  let total = 0;
  listEl.innerHTML = '';

  results.forEach((listing, idx) => {
    if (!listing) return;
    const listingId = listingIds[idx];
    const qty = cart[listingId];
    const lineTotal = parseFloat(listing.price) * qty;
    total += lineTotal;

    const img = listing.primary_photo || 'https://picsum.photos/seed/placeholder/200/200';

    const row = document.createElement('div');
    row.className = 'cart-item-row';
    row.innerHTML = `
      <img src="${img}" alt="${cartEscapeHtml(listing.title)}" />
      <div class="cart-item-details">
        <p class="cart-item-title">${cartEscapeHtml(listing.title)}</p>
        <p class="cart-item-price">${parseFloat(listing.price).toFixed(2)} ${listing.currency}</p>
        <div class="cart-item-qty-controls">
          <button data-action="dec">−</button>
          <span>${qty}</span>
          <button data-action="inc">+</button>
          <button data-action="remove" class="cart-item-remove">Usuń</button>
        </div>
      </div>
    `;

    row.querySelector('[data-action="dec"]').addEventListener('click', () => setCartQuantity(listingId, qty - 1));
    row.querySelector('[data-action="inc"]').addEventListener('click', () => {
      if (qty + 1 > listing.quantity) return;
      setCartQuantity(listingId, qty + 1);
    });
    row.querySelector('[data-action="remove"]').addEventListener('click', () => removeFromCart(listingId));

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
