const API_URL = 'http://localhost:8080';

let currentProducts = [];
let currentCart = [];

// Initialize
window.addEventListener('load', () => {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = 'index.html';
        return;
    }

    loadCategories();
    loadProducts();
    loadCart();
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

async function loadProducts(categoryId = null, search = '') {
    try {
        let url = `${API_URL}/api/products`;
        const params = new URLSearchParams();

        if (categoryId) params.append('category_id', categoryId);
        if (search) params.append('search', search);

        if (params.toString()) {
            url += '?' + params.toString();
        }

        const response = await fetch(url);
        currentProducts = await response.json() || [];

        displayProducts(currentProducts);
    } catch (error) {
        console.error('Failed to load products:', error);
    }
}

function displayProducts(products) {
    const grid = document.getElementById('productsGrid');
    grid.innerHTML = '';

    if (!products || products.length === 0) {
        grid.innerHTML = '<p style="grid-column: 1/-1; text-align: center; color: #999;">Brak produktów</p>';
        return;
    }

    products.forEach(product => {
        const stockStatus = product.stock <= 5 ? ' low' : '';
        const stockText = product.stock <= 0 ? 'Niedostępny' :
                         product.stock <= 5 ? `Tylko ${product.stock} szt.` :
                         'Dostępny';

        const card = document.createElement('div');
        card.className = 'product-card';
        card.innerHTML = `
            <img src="${product.image_url}" alt="${product.name}" class="product-image" />
            <div class="product-info">
                <div class="product-name">${product.name}</div>
                <div class="product-price">${product.price.toFixed(2)} zł</div>
                <div class="product-rating">⭐ ${product.rating.toFixed(1)} (${product.reviews_count})</div>
                <div class="product-stock${stockStatus}">${stockText}</div>
                <div class="product-buttons">
                    <button class="btn-view" onclick="showProductDetail(${product.id})">Szczegóły</button>
                    ${product.stock > 0 ?
                        `<button class="btn-add" onclick="quickAddToCart(${product.id}, 1)">Dodaj</button>` :
                        `<button class="btn-add" disabled style="opacity: 0.5;">Brak</button>`
                    }
                </div>
            </div>
        `;
        grid.appendChild(card);
    });
}

function filterProducts() {
    const categoryId = document.getElementById('categoryFilter').value;
    const search = document.getElementById('searchInput').value;
    const maxPrice = document.getElementById('priceFilter').value;

    document.getElementById('priceValue').textContent = maxPrice;

    let filtered = currentProducts;

    if (categoryId) {
        filtered = filtered.filter(p => p.category_id == categoryId);
    }

    if (search) {
        const searchLower = search.toLowerCase();
        filtered = filtered.filter(p =>
            p.name.toLowerCase().includes(searchLower) ||
            p.description.toLowerCase().includes(searchLower)
        );
    }

    filtered = filtered.filter(p => p.price <= maxPrice);

    displayProducts(filtered);
}

function resetFilters() {
    document.getElementById('categoryFilter').value = '';
    document.getElementById('searchInput').value = '';
    document.getElementById('priceFilter').value = '500';
    document.getElementById('priceValue').textContent = '500';
    loadProducts();
}

async function showProductDetail(productId) {
    try {
        const response = await fetch(`${API_URL}/api/products/${productId}`);
        const product = await response.json();

        const modal = document.getElementById('productModal');
        const detail = document.getElementById('productDetail');

        detail.innerHTML = `
            <div class="product-detail">
                <img src="${product.image_url}" alt="${product.name}" class="product-detail-image" />
                <div class="product-detail-info">
                    <h2>${product.name}</h2>
                    <div class="product-detail-price">${product.price.toFixed(2)} zł</div>

                    <div class="product-detail-meta">
                        <div class="meta-item">
                            <span class="meta-label">Ocena</span>
                            <span class="meta-value">⭐ ${product.rating.toFixed(1)}</span>
                        </div>
                        <div class="meta-item">
                            <span class="meta-label">Opinii</span>
                            <span class="meta-value">${product.reviews_count}</span>
                        </div>
                        <div class="meta-item">
                            <span class="meta-label">Dostępność</span>
                            <span class="meta-value">${product.stock > 0 ? `${product.stock} szt.` : 'Niedostępny'}</span>
                        </div>
                    </div>

                    <p class="product-detail-description">${product.description}</p>

                    ${product.stock > 0 ? `
                        <div class="quantity-selector">
                            <button class="quantity-btn" onclick="decreaseQty()">-</button>
                            <input type="number" id="detailQuantity" class="quantity-input" value="1" min="1" max="${product.stock}" />
                            <button class="quantity-btn" onclick="increaseQty(${product.stock})">+</button>
                        </div>
                        <button class="btn-add-detail" onclick="addToCartFromDetail(${product.id})">Dodaj do koszyka</button>
                    ` : `
                        <button class="btn-add-detail" disabled style="opacity: 0.5;">Niedostępny</button>
                    `}
                </div>
            </div>
        `;

        modal.classList.add('active');
    } catch (error) {
        console.error('Failed to load product detail:', error);
    }
}

function closeProductDetail() {
    document.getElementById('productModal').classList.remove('active');
}

function decreaseQty() {
    const input = document.getElementById('detailQuantity');
    if (input.value > 1) input.value = parseInt(input.value) - 1;
}

function increaseQty(max) {
    const input = document.getElementById('detailQuantity');
    if (input.value < max) input.value = parseInt(input.value) + 1;
}

async function addToCartFromDetail(productId) {
    const quantity = parseInt(document.getElementById('detailQuantity').value);
    await addToCart(productId, quantity);
    closeProductDetail();
}

async function quickAddToCart(productId, quantity) {
    await addToCart(productId, quantity);
}

async function addToCart(productId, quantity) {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/cart`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`,
            },
            body: JSON.stringify({ product_id: productId, quantity }),
        });

        if (response.ok) {
            loadCart();
            alert('Dodano do koszyka!');
        }
    } catch (error) {
        console.error('Failed to add to cart:', error);
        alert('Błąd przy dodawaniu do koszyka');
    }
}

async function loadCart() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/cart`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        currentCart = await response.json() || [];
        updateCartUI();
    } catch (error) {
        console.error('Failed to load cart:', error);
    }
}

function updateCartUI() {
    document.getElementById('cartCount').textContent = currentCart.length;

    const cartList = document.getElementById('cartList');
    cartList.innerHTML = '';

    if (currentCart.length === 0) {
        cartList.innerHTML = '<div class="cart-empty">Koszyk jest pusty</div>';
        document.getElementById('cartTotal').textContent = '0,00 zł';
        return;
    }

    let total = 0;
    currentCart.forEach(item => {
        total += item.product.price * item.quantity;

        const cartItem = document.createElement('div');
        cartItem.className = 'cart-item';
        cartItem.innerHTML = `
            <img src="${item.product.image_url}" alt="${item.product.name}" class="cart-item-image" />
            <div class="cart-item-info">
                <h4>${item.product.name}</h4>
                <p>${item.product.price.toFixed(2)} zł × ${item.quantity}</p>
            </div>
            <div class="cart-item-quantity">
                <button class="quantity-sm" onclick="updateCartQuantity(${item.id}, ${item.quantity - 1})">-</button>
                <span>${item.quantity}</span>
                <button class="quantity-sm" onclick="updateCartQuantity(${item.id}, ${item.quantity + 1})">+</button>
            </div>
            <button class="cart-item-remove" onclick="removeFromCart(${item.id})">✕</button>
        `;
        cartList.appendChild(cartItem);
    });

    document.getElementById('cartTotal').textContent = total.toFixed(2) + ' zł';
}

async function updateCartQuantity(cartItemId, newQuantity) {
    if (newQuantity <= 0) {
        await removeFromCart(cartItemId);
        return;
    }

    try {
        const token = localStorage.getItem('access_token');
        await fetch(`${API_URL}/api/cart/${cartItemId}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`,
            },
            body: JSON.stringify({ quantity: newQuantity }),
        });

        loadCart();
    } catch (error) {
        console.error('Failed to update cart:', error);
    }
}

async function removeFromCart(cartItemId) {
    try {
        const token = localStorage.getItem('access_token');
        await fetch(`${API_URL}/api/cart/${cartItemId}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` },
        });

        loadCart();
    } catch (error) {
        console.error('Failed to remove from cart:', error);
    }
}

function openCart() {
    document.getElementById('cartModal').classList.add('active');
}

function closeCart() {
    document.getElementById('cartModal').classList.remove('active');
}

async function handleCheckout() {
    if (currentCart.length === 0) {
        alert('Koszyk jest pusty');
        return;
    }
    window.location.href = 'checkout.html';
}

function logout() {
    localStorage.clear();
    window.location.href = '../index.html';
}
