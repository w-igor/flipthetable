const API_URL = 'http://localhost:8080';

let currentShop = null;
let categories = [];
let currentEditingListingId = null;

// Initialize
window.addEventListener('load', async () => {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = '../index.html';
        return;
    }

    await loadShop();
    await loadCategories();
    loadListings();
    loadOrders();
});

async function loadShop() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/auth/me`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        if (!response.ok) {
            showNoShopMessage();
            return;
        }

        const user = await response.json();

        if (!user.is_seller) {
            showNoShopMessage();
            return;
        }

        document.getElementById('shopName').textContent = user.username + ' Store';
    } catch (error) {
        console.error('Failed to load shop:', error);
        showNoShopMessage();
    }
}

function showNoShopMessage() {
    const content = document.querySelector('.seller-content');
    content.innerHTML = `
        <div style="text-align: center; padding: 50px;">
            <h2>Nie jesteś sprzedawcą</h2>
            <p style="color: #666; margin: 15px 0;">Aby sprzedawać produkty, musisz najpierw stać się sprzedawcą.</p>
        </div>
    `;
}

async function loadCategories() {
    try {
        const response = await fetch(`${API_URL}/api/categories`);
        categories = await response.json() || [];

        const select = document.getElementById('listingCategory');
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
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/seller/products`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        if (!response.ok) return;

        const listings = await response.json() || [];

        const list = document.getElementById('listingsList');
        if (listings.length === 0) {
            list.innerHTML = '<p style="grid-column: 1/-1; text-align: center; color: #999;">Brak produktów</p>';
            document.getElementById('statListings').textContent = '0';
            return;
        }

        document.getElementById('statListings').textContent = listings.length;
        list.innerHTML = listings.map(l => `
            <div class="listing-item">
                <img src="${l.image_url || 'https://via.placeholder.com/200'}" alt="${l.name}" class="listing-item-image" />
                <div class="listing-item-info">
                    <div class="listing-item-name">${l.name}</div>
                    <div class="listing-item-meta">
                        <span class="listing-item-price">${l.price.toFixed(2)} zł</span>
                        <span class="listing-item-quantity ${l.stock <= 5 ? 'low' : ''}">${l.stock} szt.</span>
                    </div>
                    <div class="listing-item-actions">
                        <button class="btn-edit" onclick="editListing('${l.id}')">Edytuj</button>
                        <button class="btn-delete" onclick="deleteListing('${l.id}')">Usuń</button>
                    </div>
                </div>
            </div>
        `).join('');
    } catch (error) {
        console.error('Failed to load listings:', error);
    }
}

async function loadOrders() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/seller/orders`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        if (!response.ok) return;

        const orders = await response.json() || [];

        const list = document.getElementById('sellerOrdersList');
        if (orders.length === 0) {
            list.innerHTML = '<p style="text-align: center; color: #999;">Brak zamówień</p>';
            document.getElementById('statOrders').textContent = '0';
            return;
        }

        const statusLabels = {
            'pending': 'Oczekujące',
            'paid': 'Zapłacone',
            'processing': 'Przetwarzane',
            'shipped': 'Wysłane',
            'delivered': 'Dostarczone',
            'cancelled': 'Anulowane',
        };

        document.getElementById('statOrders').textContent = orders.length;

        let totalSales = 0;
        orders.forEach(o => {
            if (o.status !== 'cancelled') {
                totalSales += o.total_amount;
            }
        });
        document.getElementById('statSales').textContent = totalSales.toFixed(2) + ' zł';

        list.innerHTML = orders.map(o => `
            <div class="order-row">
                <div class="order-row-info">
                    <span class="order-row-label">Zamówienie</span>
                    <span class="order-row-value">#${o.id.substring(0, 8).toUpperCase()}</span>
                </div>
                <div class="order-row-info">
                    <span class="order-row-label">Data</span>
                    <span class="order-row-value">${new Date(o.created_at).toLocaleDateString('pl-PL')}</span>
                </div>
                <div class="order-row-info">
                    <span class="order-row-label">Status</span>
                    <span class="order-row-value">${statusLabels[o.status]}</span>
                </div>
                <div class="order-row-info">
                    <span class="order-row-label">Razem</span>
                    <span class="order-row-value">${o.total_amount.toFixed(2)} zł</span>
                </div>
            </div>
        `).join('');
    } catch (error) {
        console.error('Failed to load orders:', error);
    }
}

function switchTab(tabName) {
    document.querySelectorAll('.tab-content').forEach(tab => {
        tab.classList.remove('active');
    });

    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
    });

    const tab = document.getElementById(tabName + 'Tab');
    if (tab) tab.classList.add('active');

    event.target?.classList.add('active');
}

function openAddListingModal() {
    currentEditingListingId = null;
    document.getElementById('listingModalTitle').textContent = 'Dodaj produkt';
    document.getElementById('listingForm').reset();
    document.getElementById('listingModal').classList.add('active');
}

function closeListingModal() {
    document.getElementById('listingModal').classList.remove('active');
}

async function editListing(listingId) {
    try {
        const response = await fetch(`${API_URL}/api/listings/${listingId}`);
        const listing = await response.json();

        currentEditingListingId = listingId;
        document.getElementById('listingModalTitle').textContent = 'Edytuj produkt';
        document.getElementById('listingTitle').value = listing.title;
        document.getElementById('listingCategory').value = listing.category_id || '';
        document.getElementById('listingDescription').value = listing.description;
        document.getElementById('listingPrice').value = listing.price;
        document.getElementById('listingQuantity').value = listing.quantity;
        document.getElementById('listingPhotoUrl').value = listing.photos?.[0]?.url || '';

        document.getElementById('listingModal').classList.add('active');
    } catch (error) {
        alert('Błąd przy ładowaniu produktu');
        console.error(error);
    }
}

async function handleListingSubmit(event) {
    event.preventDefault();

    const listing = {
        title: document.getElementById('listingTitle').value,
        category_id: document.getElementById('listingCategory').value,
        description: document.getElementById('listingDescription').value,
        price: parseFloat(document.getElementById('listingPrice').value),
        quantity: parseInt(document.getElementById('listingQuantity').value),
        photos: document.getElementById('listingPhotoUrl').value ? [document.getElementById('listingPhotoUrl').value] : [],
    };

    try {
        const token = localStorage.getItem('access_token');
        let url, method;

        if (currentEditingListingId) {
            url = `${API_URL}/api/seller/products/${currentEditingListingId}`;
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
            body: JSON.stringify(listing),
        });

        if (response.ok) {
            closeListingModal();
            await loadListings();
            alert(currentEditingListingId ? 'Produkt zaktualizowany' : 'Produkt dodany');
        } else {
            alert('Błąd przy zapisywaniu produktu');
        }
    } catch (error) {
        alert('Błąd przy zapisywaniu produktu');
        console.error(error);
    }
}

async function deleteListing(listingId) {
    if (!confirm('Czy na pewno chcesz usunąć ten produkt?')) return;

    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/seller/products/${listingId}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` },
        });

        if (response.ok) {
            await loadListings();
            alert('Produkt usunięty');
        }
    } catch (error) {
        alert('Błąd przy usuwaniu produktu');
        console.error(error);
    }
}

function logout() {
    localStorage.clear();
    window.location.href = '../index.html';
}
