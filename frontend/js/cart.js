// Shopping Cart Management
// Handles local cart state, persistence, and display updates

const CART_API_URL = window.API_URL || 'http://localhost:8080';
const CART_STORAGE_KEY = 'cart';

// Generates a unique key for a cart entry (listing + optional variant)
function cartKey(listingId, variantSkuId) {
  return variantSkuId ? `${listingId}__${variantSkuId}` : listingId;
}

// Retrieves the shopping cart from localStorage
function getCart() {
  try {
    return JSON.parse(localStorage.getItem(CART_STORAGE_KEY)) || {};
  } catch (err) {
    return {};
  }
}

// Persists cart to localStorage and updates the badge
function saveCart(cart) {
  localStorage.setItem(CART_STORAGE_KEY, JSON.stringify(cart));
  updateCartBadge();
}

// Adds an item to the cart (quantity is additive if item already exists)
// variantSkuId is null for listings without variants
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

// Updates the quantity of an item in the cart (or removes if quantity <= 0)
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

// Removes an item from the cart and updates display
function removeFromCart(key) {
  const cart = getCart();
  delete cart[key];
  saveCart(cart);
  renderCartDrawer();
}

// Calculates the total number of items in the cart
function getCartCount() {
  const cart = getCart();
  return Object.values(cart).reduce((sum, entry) => sum + entry.quantity, 0);
}

// Updates the cart badge with the current item count
function updateCartBadge() {
  const badge = document.getElementById('cartCount');
  if (badge) badge.textContent = getCartCount();
}

// Safely escapes HTML to prevent XSS
function cartEscapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// Resolves price, quantity, and label for a cart line from listing data
// Handles both simple products and variant-based products
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

// Renders the shopping cart drawer with current items and total
// Fetches listing details and calculates totals per item
async function renderCartDrawer() {
  const listEl = document.getElementById('cartItemsList');
  const totalEl = document.getElementById('cartTotalAmount');
  if (!listEl) return;

  const cart = getCart();
  const entries = Object.entries(cart);

  // Handle empty cart
  if (entries.length === 0) {
    listEl.innerHTML = `<p class="cart-empty">${t('cart.empty')}</p>`;
    if (totalEl) totalEl.textContent = '0.00 PLN';
    return;
  }

  // Show loading state while fetching listing data
  listEl.innerHTML = `<p class="cart-loading">${t('common.loading')}</p>`;

  // Batch fetch all listing details
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

  // Render each cart item
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
          <button data-action="remove" class="cart-item-remove">${t('cart.remove')}</button>
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

  // Update cart total
  if (totalEl) totalEl.textContent = `${total.toFixed(2)} PLN`;
}

// Toggles the cart drawer visibility
function toggleCartDrawer(forceOpen) {
  const drawer = document.getElementById('cartDrawer');
  const overlay = document.getElementById('cartOverlay');
  if (!drawer || !overlay) return;

  const shouldOpen = forceOpen !== undefined ? forceOpen : !drawer.classList.contains('open');
  drawer.classList.toggle('open', shouldOpen);
  overlay.classList.toggle('open', shouldOpen);

  // Render cart when opening
  if (shouldOpen) renderCartDrawer();
}

// Initializes the shopping cart widget on page load
function initCartWidget() {
  updateCartBadge();

  const toggleBtn = document.getElementById('cartToggleBtn');
  const closeBtn = document.getElementById('closeCartBtn');
  const overlay = document.getElementById('cartOverlay');

  if (toggleBtn) toggleBtn.addEventListener('click', () => toggleCartDrawer());
  if (closeBtn) closeBtn.addEventListener('click', () => toggleCartDrawer(false));
  if (overlay) overlay.addEventListener('click', () => toggleCartDrawer(false));
}

// Initialize cart widget when DOM is ready
document.addEventListener('DOMContentLoaded', initCartWidget);
