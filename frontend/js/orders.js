const API_URL = 'http://localhost:8080';

let allOrders = [];

// Initialize
window.addEventListener('load', () => {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = '../index.html';
        return;
    }

    loadOrders();
    loadStats();
});

async function loadOrders() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/orders`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        allOrders = await response.json() || [];
        displayOrders(allOrders);
    } catch (error) {
        console.error('Failed to load orders:', error);
        document.getElementById('ordersList').innerHTML = '<div class="empty-state"><h3>Błąd przy ładowaniu zamówień</h3></div>';
    }
}

async function loadStats() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/orders/stats/summary`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        const stats = await response.json();

        document.getElementById('totalOrders').textContent = stats.total_orders || 0;
        document.getElementById('totalSpent').textContent = (stats.total_spent || 0).toFixed(2) + ' zł';
        document.getElementById('pendingOrders').textContent = stats.pending_orders || 0;
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

function displayOrders(orders) {
    const container = document.getElementById('ordersList');

    if (!orders || orders.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <h3>Brak zamówień</h3>
                <p>Nie złożyłeś jeszcze żadnego zamówienia</p>
                <a href="shop.html" style="color: #667eea; text-decoration: none; margin-top: 10px; display: inline-block;">Przejdź do sklepu →</a>
            </div>
        `;
        return;
    }

    container.innerHTML = '';

    orders.forEach(order => {
        const orderCard = document.createElement('div');
        orderCard.className = 'order-card';
        orderCard.onclick = () => toggleOrderExpand(orderCard);

        const itemsCount = order.items ? order.items.length : 0;
        const createdDate = new Date(order.created_at).toLocaleDateString('pl-PL');

        let itemsHTML = '';
        if (order.items && order.items.length > 0) {
            order.items.forEach(item => {
                itemsHTML += `
                    <div class="order-item-row">
                        <img src="${item.product.image_url}" alt="${item.product.name}" class="order-item-image" />
                        <div class="order-item-info">
                            <h4>${item.product.name}</h4>
                            <p>Ilość: ${item.quantity}</p>
                        </div>
                        <div class="order-item-price">${(item.price_at_purchase * item.quantity).toFixed(2)} zł</div>
                    </div>
                `;
            });
        }

        orderCard.innerHTML = `
            <div class="order-card-header">
                <div class="order-info-item">
                    <span class="order-info-label">Zamówienie</span>
                    <span class="order-info-value order-id">#${order.id}</span>
                </div>
                <div class="order-info-item">
                    <span class="order-info-label">Data</span>
                    <span class="order-info-value order-date">${createdDate}</span>
                </div>
                <div class="order-info-item">
                    <span class="order-info-label">Razem</span>
                    <span class="order-info-value order-total">${order.total_price.toFixed(2)} zł</span>
                </div>
                <div class="order-info-item">
                    <span class="order-info-label">Status</span>
                    <span class="status-badge status-${order.status}">${getStatusLabel(order.status)}</span>
                </div>
            </div>

            <div class="order-card-body">
                <div class="order-items">
                    <h3>Produkty (${itemsCount})</h3>
                    ${itemsHTML}
                </div>
                <div class="detail-section">
                    <h3>Szczegóły</h3>
                    <div class="detail-row">
                        <span>Status zamówienia:</span>
                        <strong>${getStatusLabel(order.status)}</strong>
                    </div>
                    <div class="detail-row">
                        <span>Data zamówienia:</span>
                        <strong>${createdDate}</strong>
                    </div>
                    <div class="detail-row">
                        <span>Ostatnia aktualizacja:</span>
                        <strong>${new Date(order.updated_at).toLocaleDateString('pl-PL')}</strong>
                    </div>
                    <div class="detail-row">
                        <span>Razem do zapłaty:</span>
                        <strong>${order.total_price.toFixed(2)} zł</strong>
                    </div>
                </div>
            </div>

            <div class="order-card-footer">
                <div>Kliknij aby rozwinąć szczegóły</div>
                <div class="order-actions">
                    <button class="btn-view-detail" onclick="event.stopPropagation(); viewOrderDetail(${order.id})">Szczegóły</button>
                    ${order.status === 'shipped' ? `<button class="btn-track" onclick="event.stopPropagation(); alert('Śledzenie dostępne dla wysłanych zamówień')">Śledź paczkę</button>` : ''}
                </div>
            </div>
        `;

        container.appendChild(orderCard);
    });
}

function toggleOrderExpand(card) {
    // Remove expanded class from all cards
    document.querySelectorAll('.order-card').forEach(c => {
        if (c !== card) c.classList.remove('expanded');
    });
    card.classList.toggle('expanded');
}

function getStatusLabel(status) {
    const labels = {
        'pending': 'Oczekujące',
        'confirmed': 'Potwierdzone',
        'shipped': 'Wysłane',
        'delivered': 'Dostarczone',
        'cancelled': 'Anulowane'
    };
    return labels[status] || status;
}

async function viewOrderDetail(orderId) {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/orders/${orderId}`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        if (!response.ok) throw new Error('Order not found');

        const order = await response.json();
        displayOrderDetailModal(order);
    } catch (error) {
        console.error('Failed to load order detail:', error);
        alert('Błąd przy ładowaniu szczegółów zamówienia');
    }
}

function displayOrderDetailModal(order) {
    const modal = document.getElementById('orderModal');
    const detail = document.getElementById('orderDetail');

    let itemsHTML = '';
    if (order.items && order.items.length > 0) {
        order.items.forEach(item => {
            itemsHTML += `
                <div class="detail-row">
                    <span>${item.product.name} × ${item.quantity}</span>
                    <strong>${(item.price_at_purchase * item.quantity).toFixed(2)} zł</strong>
                </div>
            `;
        });
    }

    detail.innerHTML = `
        <h2>Zamówienie #${order.id}</h2>

        <div class="detail-section">
            <h3>Status</h3>
            <span class="status-badge status-${order.status}">${getStatusLabel(order.status)}</span>
        </div>

        <div class="detail-section">
            <h3>Produkty</h3>
            ${itemsHTML}
        </div>

        <div class="detail-section">
            <h3>Razem</h3>
            <div class="detail-row">
                <span>Wartość zamówienia:</span>
                <strong>${order.total_price.toFixed(2)} zł</strong>
            </div>
        </div>

        <div class="detail-section">
            <h3>Informacje</h3>
            <div class="detail-row">
                <span>Data zamówienia:</span>
                <strong>${new Date(order.created_at).toLocaleDateString('pl-PL')}</strong>
            </div>
            <div class="detail-row">
                <span>Ostatnia zmiana:</span>
                <strong>${new Date(order.updated_at).toLocaleDateString('pl-PL')}</strong>
            </div>
        </div>
    `;

    modal.classList.add('active');
}

function closeOrderDetail() {
    document.getElementById('orderModal').classList.remove('active');
}

document.getElementById('searchFilter').addEventListener('keyup', filterOrders);

function filterOrders() {
    const searchQuery = document.getElementById('searchFilter').value.toLowerCase();
    const statusFilter = document.getElementById('statusFilter').value;

    let filtered = allOrders.filter(order => {
        const matchesSearch = order.id.toString().includes(searchQuery);
        const matchesStatus = !statusFilter || order.status === statusFilter;
        return matchesSearch && matchesStatus;
    });

    displayOrders(filtered);
}
