const API_URL = 'http://localhost:8080';

let allListings = [];
let cart = [];

// Initialize
window.addEventListener('load', () => {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = '../index.html';
        return;
    }

    loadCategories();
    loadListings();
    loadCart();
    checkIfSeller();
});

async function loadCategories() {
    try {
        const response = await fetch(`${API_URL}/api/categories`);
        const categories = await response.json();

        const select = document.getElementById('categoryFilter');
        categories.forEach(cat => {
            const option = document.createElement('option');
            option.value = cat.id;
            option.textContent = cat.name;
            select.appendChild(option);
        });
    } catch (error) {
        console.error('Failed to load categories:', error);
    }
}

async function loadListings() {
    try {
        const response = await fetch(`${API_URL}/api/listings`);
        allListings = await response.json() || [];
        displayListings(allListings);
    } catch (error) {
        console.error('Failed to load listings:', error);
    }
}

function displayListings(listings) {
    const grid = document.getElementById('listingsGrid');
    grid.innerHTML = '';

    if (!listings || listings.length === 0) {
        grid.innerHTML = '<p style="grid-column: 1/-1; text-align: center; color: #999;">Brak produktów</p>';
        return;
    }

    listings.forEach(listing => {
        const stockStatus = listing.quantity <= 5 ? ' low' : '';
        const stockText = listing.quantity <= 0 ? 'Niedostępny' :
                         listing.quantity <= 5 ? `Tylko ${listing.quantity} szt.` :
                         'Dostępny';

        const primaryPhoto = listing.photos?.find(p => p.is_primary)?.url || 'https://via.placeholder.com/200?text=No+Image';

        const card = document.createElement('div');
        card.className = 'product-card';
        card.innerHTML = `
            <img src="${primaryPhoto}" alt="${listing.title}" class="product-image" />
            <div class="product-info">
                <div class="product-name">${listing.title}</div>
                <div class="product-price">${listing.price.toFixed(2)} zł</div>
                <div class="product-rating">⭐ ${listing.avg_rating?.toFixed(1) || 'Brak'}</div>
                <div class="product-stock${stockStatus}">${stockText}</div>
                <div class="product-buttons">
                    <button class="btn-view" onclick="showListingDetail('${listing.id}')">Szczegóły</button>
                    ${listing.quantity > 0 ?
                        `<button class="btn-add" onclick="quickAddToCart('${listing.id}', 1)">Dodaj</button>` :
                        `<button class="btn-add" disabled style="opacity: 0.5;">Brak</button>`
                    }
                </div>
            </div>
        `;
        grid.appendChild(card);
    });
}

function filterListings() {
    const categoryId = document.getElementById('categoryFilter').value;
    const search = document.getElementById('searchInput').value;

    let filtered = allListings;

    if (categoryId) {
        filtered = filtered.filter(l => l.category_id === categoryId);
    }

    if (search) {
        const searchLower = search.toLowerCase();
        filtered = filtered.filter(l =>
            l.title.toLowerCase().includes(searchLower) ||
            l.description.toLowerCase().includes(searchLower)
        );
    }

    displayListings(filtered);
}

function resetFilters() {
    document.getElementById('categoryFilter').value = '';
    document.getElementById('searchInput').value = '';
    loadListings();
}

async function showListingDetail(listingId) {
    try {
        const response = await fetch(`${API_URL}/api/listings/${listingId}`);
        const listing = await response.json();

        const modal = document.getElementById('listingModal');
        const detail = document.getElementById('listingDetail');

        const primaryPhoto = listing.photos?.find(p => p.is_primary)?.url || 'https://via.placeholder.com/200';

        detail.innerHTML = `
            <div class="listing-detail">
                <img src="${primaryPhoto}" alt="${listing.title}" class="listing-detail-image" />
                <div class="listing-detail-info">
                    <h2>${listing.title}</h2>
                    <div class="listing-detail-price">${listing.price.toFixed(2)} zł</div>

                    <div class="listing-detail-meta">
                        <div class="meta-item">
                            <span class="meta-label">Ocena</span>
                            <span class="meta-value">⭐ ${listing.avg_rating?.toFixed(1) || 'Brak'}</span>
                        </div>
                        <div class="meta-item">
                            <span class="meta-label">Dostępność</span>
                            <span class="meta-value">${listing.quantity > 0 ? `${listing.quantity} szt.` : 'Niedostępny'}</span>
                        </div>
                    </div>

                    <p class="listing-detail-description">${listing.description}</p>

                    ${listing.quantity > 0 ? `
                        <div class="quantity-selector">
                            <button class="quantity-btn" onclick="decreaseQty()">-</button>
                            <input type="number" id="detailQuantity" class="quantity-input" value="1" min="1" max="${listing.quantity}" />
                            <button class="quantity-btn" onclick="increaseQty(${listing.quantity})">+</button>
                        </div>
                        <button class="btn-add-detail" onclick="addToCartFromDetail('${listing.id}')">Dodaj do koszyka</button>
                    ` : `
                        <button class="btn-add-detail" disabled style="opacity: 0.5;">Niedostępny</button>
                    `}
                </div>
            </div>
        `;

        modal.classList.add('active');
    } catch (error) {
        console.error('Failed to load listing detail:', error);
    }
}

function closeListingDetail() {
    document.getElementById('listingModal').classList.remove('active');
}

function decreaseQty() {
    const input = document.getElementById('detailQuantity');
    if (input.value > 1) input.value = parseInt(input.value) - 1;
}

function increaseQty(max) {
    const input = document.getElementById('detailQuantity');
    if (input.value < max) input.value = parseInt(input.value) + 1;
}

async function addToCartFromDetail(listingId) {
    const quantity = parseInt(document.getElementById('detailQuantity').value);
    await addToCart(listingId, quantity);
    closeListingDetail();
}

async function quickAddToCart(listingId, quantity) {
    await addToCart(listingId, quantity);
}

async function addToCart(listingId, quantity) {
    const existingItem = cart.find(item => item.listing_id === listingId);

    if (existingItem) {
        existingItem.quantity += quantity;
    } else {
        cart.push({ listing_id: listingId, quantity });
    }

    localStorage.setItem('cart', JSON.stringify(cart));
    updateCartUI();
    alert('Dodano do koszyka!');
}

async function loadCart() {
    const savedCart = localStorage.getItem('cart');
    if (savedCart) {
        cart = JSON.parse(savedCart);
    }
    updateCartUI();
}

function updateCartUI() {
    document.getElementById('cartCount').textContent = cart.length;

    const cartList = document.getElementById('cartList');
    cartList.innerHTML = '';

    if (cart.length === 0) {
        cartList.innerHTML = '<div class="cart-empty">Koszyk jest pusty</div>';
        document.getElementById('cartTotal').textContent = '0,00 zł';
        return;
    }

    let total = 0;
    cart.forEach(item => {
        const listing = allListings.find(l => l.id === item.listing_id);
        if (listing) {
            total += listing.price * item.quantity;

            const cartItem = document.createElement('div');
            cartItem.className = 'cart-item';
            cartItem.innerHTML = `
                <img src="${listing.photos?.[0]?.url || 'https://via.placeholder.com/200'}" alt="${listing.title}" class="cart-item-image" />
                <div class="cart-item-info">
                    <h4>${listing.title}</h4>
                    <p>${listing.price.toFixed(2)} zł × ${item.quantity}</p>
                </div>
                <div class="cart-item-quantity">
                    <button class="quantity-sm" onclick="updateCartQuantity('${listing.id}', ${item.quantity - 1})">-</button>
                    <span>${item.quantity}</span>
                    <button class="quantity-sm" onclick="updateCartQuantity('${listing.id}', ${item.quantity + 1})">+</button>
                </div>
                <button class="cart-item-remove" onclick="removeFromCart('${listing.id}')">✕</button>
            `;
            cartList.appendChild(cartItem);
        }
    });

    document.getElementById('cartTotal').textContent = total.toFixed(2) + ' zł';
}

function updateCartQuantity(listingId, newQuantity) {
    if (newQuantity <= 0) {
        removeFromCart(listingId);
        return;
    }

    const item = cart.find(i => i.listing_id === listingId);
    if (item) {
        item.quantity = newQuantity;
        localStorage.setItem('cart', JSON.stringify(cart));
        updateCartUI();
    }
}

function removeFromCart(listingId) {
    cart = cart.filter(i => i.listing_id !== listingId);
    localStorage.setItem('cart', JSON.stringify(cart));
    updateCartUI();
}

function openCart() {
    document.getElementById('cartModal').classList.add('active');
}

function closeCart() {
    document.getElementById('cartModal').classList.remove('active');
}

async function handleCheckout() {
    if (cart.length === 0) {
        alert('Koszyk jest pusty');
        return;
    }

    // Redirect to checkout
    window.location.href = 'checkout.html';
}

async function checkIfSeller() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/auth/me`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        if (response.ok) {
            const user = await response.json();
            if (user.is_seller) {
                document.getElementById('sellerLink').style.display = 'inline-block';
            }
        }
    } catch (error) {
        // Not a seller
    }
}

function logout() {
    localStorage.clear();
    window.location.href = '../index.html';
}
