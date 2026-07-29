const API_URL = 'http://localhost:8080';
let currentEditingProductId = null;
let categories = [];

// Initialize
window.addEventListener('load', async () => {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = '../index.html';
        return;
    }

    await loadCategories();
    await loadSellerProfile();
    await loadSellerStats();
    switchTab('dashboard');
});

async function loadCategories() {
    try {
        const response = await fetch(`${API_URL}/api/categories`);
        categories = await response.json() || [];

        const select = document.getElementById('productCategory');
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

async function loadSellerProfile() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/seller/profile`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        if (!response.ok) {
            // User is not a seller yet
            showNoSellerMessage();
            return;
        }

        const profile = await response.json();

        document.getElementById('sellerName').textContent = profile.seller_name || 'Sprzedawca';
        document.getElementById('sellerNameInput').value = profile.seller_name || '';
        document.getElementById('sellerDescriptionInput').value = profile.seller_description || '';
        document.getElementById('profileEmail').textContent = profile.email;
        document.getElementById('profileJoined').textContent = new Date(profile.joined_at).toLocaleDateString('pl-PL');

        const statusEl = document.getElementById('profileStatus');
        if (profile.seller_verified) {
            statusEl.textContent = 'Zweryfikowany';
            statusEl.className = 'badge verified';
        }

        // Load products and stats
        await loadSellerProducts();
        await loadSellerOrders();
    } catch (error) {
        console.error('Failed to load seller profile:', error);
    }
}

function showNoSellerMessage() {
    const content = document.querySelector('.seller-content');
    content.innerHTML = `
        <div style="text-align: center; padding: 50px;">
            <h2>Nie jesteś sprzedawcą</h2>
            <p style="color: #666; margin: 15px 0;">Aby mieć dostęp do panelu sprzedawcy, musisz się zarejestrować jako sprzedawca.</p>
            <button class="btn-primary" onclick="registerAsSeller()">Zarejestruj się jako sprzedawca</button>
        </div>
    `;
}

async function registerAsSeller() {
    const sellerName = prompt('Podaj nazwę swojego sklepu:');
    if (!sellerName) return;

    const description = prompt('Opis sklepu (opcjonalnie):') || '';

    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/seller/register`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`,
            },
            body: JSON.stringify({
                seller_name: sellerName,
                seller_description: description,
            }),
        });

        if (response.ok) {
            alert('Zarejestrowano jako sprzedawca!');
            location.reload();
        }
    } catch (error) {
        alert('Błąd przy rejestracji');
        console.error(error);
    }
}

async function loadSellerStats() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/seller/stats`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        const stats = await response.json();

        document.getElementById('statProducts').textContent = stats.total_products;
        document.getElementById('statOrders').textContent = stats.total_orders;
        document.getElementById('statSalesMonth').textContent = stats.sales_this_month.toFixed(2) + ' zł';
        document.getElementById('statInventory').textContent = stats.inventory_value.toFixed(2) + ' zł';

        document.getElementById('analyticsTotalSales').textContent = stats.total_sales.toFixed(2) + ' zł';
        document.getElementById('analyticsRating').textContent = stats.average_rating.toFixed(1) + '/5 ⭐';
        document.getElementById('analyticsMonthOrders').textContent = stats.orders_this_month;
        document.getElementById('analyticsMonthSales').textContent = stats.sales_this_month.toFixed(2) + ' zł';

        // Top product
        if (stats.top_product) {
            document.getElementById('topProductInfo').innerHTML = `<strong>${stats.top_product}</strong>`;
        }
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

async function loadSellerProducts() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/seller/products`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        const products = await response.json() || [];

        const list = document.getElementById('productsList');
        if (products.length === 0) {
            list.innerHTML = '<p style="grid-column: 1/-1; text-align: center; color: #999;">Brak produktów. <a href="#" onclick="openAddProductModal(); return false;">Dodaj pierwszy produkt</a></p>';
            return;
        }

        list.innerHTML = products.map(p => `
            <div class="product-item">
                <img src="${p.image_url}" alt="${p.name}" class="product-item-image" />
                <div class="product-item-info">
                    <div class="product-item-name">${p.name}</div>
                    <div class="product-item-meta">
                        <span class="product-item-price">${p.price.toFixed(2)} zł</span>
                        <span class="product-item-stock ${p.stock <= 5 ? 'low' : ''}">${p.stock} szt.</span>
                    </div>
                    <div class="product-item-actions">
                        <button class="btn-edit" onclick="editProduct(${p.id})">Edytuj</button>
                        <button class="btn-delete" onclick="deleteProduct(${p.id})">Usuń</button>
                    </div>
                </div>
            </div>
        `).join('');
    } catch (error) {
        console.error('Failed to load products:', error);
    }
}

async function loadSellerOrders() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/seller/orders`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        const orders = await response.json() || [];

        const list = document.getElementById('sellerOrdersList');
        if (orders.length === 0) {
            list.innerHTML = '<p style="text-align: center; color: #999;">Brak zamówień</p>';
            return;
        }

        const statusLabels = {
            'pending': 'Oczekujące',
            'confirmed': 'Potwierdzone',
            'shipped': 'Wysłane',
            'delivered': 'Dostarczone',
            'cancelled': 'Anulowane',
        };

        list.innerHTML = orders.map(o => `
            <div class="order-row">
                <div class="order-row-info">
                    <span class="order-row-label">Zamówienie</span>
                    <span class="order-row-value">#${o.id}</span>
                </div>
                <div class="order-row-info">
                    <span class="order-row-label">Data</span>
                    <span class="order-row-value">${new Date(o.created_at).toLocaleDateString('pl-PL')}</span>
                </div>
                <div class="order-row-info">
                    <span class="order-row-label">Status</span>
                    <span class="order-row-value">${statusLabels[o.status]}</span>
                </div>
                <div class="order-row-actions">
                    <button class="btn-view" onclick="viewOrder(${o.id})">Szczegóły</button>
                </div>
            </div>
        `).join('');
    } catch (error) {
        console.error('Failed to load seller orders:', error);
    }
}

function switchTab(tabName) {
    // Hide all tabs
    document.querySelectorAll('.tab-content').forEach(tab => {
        tab.classList.remove('active');
    });

    // Deactivate all nav items
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
    });

    // Show selected tab
    const tab = document.getElementById(tabName + 'Tab');
    if (tab) tab.classList.add('active');

    // Activate nav item
    event.target?.classList.add('active');
}

function openAddProductModal() {
    currentEditingProductId = null;
    document.getElementById('productModalTitle').textContent = 'Dodaj produkt';
    document.getElementById('productForm').reset();
    document.getElementById('productModal').classList.add('active');
}

function closeProductModal() {
    document.getElementById('productModal').classList.remove('active');
}

async function editProduct(productId) {
    // Load product data
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/products/${productId}`);
        const product = await response.json();

        currentEditingProductId = productId;
        document.getElementById('productModalTitle').textContent = 'Edytuj produkt';
        document.getElementById('productName').value = product.name;
        document.getElementById('productCategory').value = product.category_id;
        document.getElementById('productDescription').value = product.description;
        document.getElementById('productPrice').value = product.price;
        document.getElementById('productStock').value = product.stock;
        document.getElementById('productImage').value = product.image_url;

        document.getElementById('productModal').classList.add('active');
    } catch (error) {
        alert('Błąd przy ładowaniu produktu');
        console.error(error);
    }
}

async function handleProductSubmit(event) {
    event.preventDefault();

    const product = {
        name: document.getElementById('productName').value,
        category_id: parseInt(document.getElementById('productCategory').value),
        description: document.getElementById('productDescription').value,
        price: parseFloat(document.getElementById('productPrice').value),
        stock: parseInt(document.getElementById('productStock').value),
        image_url: document.getElementById('productImage').value || 'https://via.placeholder.com/200',
    };

    try {
        const token = localStorage.getItem('access_token');
        let url, method;

        if (currentEditingProductId) {
            url = `${API_URL}/api/seller/products/${currentEditingProductId}`;
            method = 'PUT';
        } else {
            url = `${API_URL}/api/seller/products`;
            method = 'POST';
        }

        const response = await fetch(url, {
            method,
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`,
            },
            body: JSON.stringify(product),
        });

        if (response.ok) {
            closeProductModal();
            await loadSellerProducts();
            await loadSellerStats();
            notificationManager.showToast(
                currentEditingProductId ? 'Produkt zaktualizowany' : 'Produkt dodany',
                'Zmiany zostały zapisane',
                'success'
            );
        }
    } catch (error) {
        alert('Błąd przy zapisywaniu produktu');
        console.error(error);
    }
}

async function deleteProduct(productId) {
    if (!confirm('Czy na pewno chcesz usunąć ten produkt?')) return;

    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/seller/products/${productId}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` },
        });

        if (response.ok) {
            await loadSellerProducts();
            await loadSellerStats();
            notificationManager.showToast('Produkt usunięty', 'Produkt został usunięty ze sklepu', 'success');
        }
    } catch (error) {
        alert('Błąd przy usuwaniu produktu');
        console.error(error);
    }
}

async function viewOrder(orderId) {
    alert(`Szczegóły zamówienia #${orderId} - feature coming soon`);
}

// Profile form submission
document.getElementById('profileForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    // TODO: Implement profile update
    alert('Aktualizacja profilu - feature coming soon');
});

function logout() {
    localStorage.clear();
    window.location.href = '../index.html';
}
