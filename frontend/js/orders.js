const API_URL = 'http://localhost:8080';

// Initialize
window.addEventListener('load', () => {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = '../index.html';
        return;
    }

    loadOrders();
});

async function loadOrders() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/orders`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        const orders = await response.json() || [];
        displayOrders(orders);
    } catch (error) {
        console.error('Failed to load orders:', error);
        document.getElementById('ordersList').innerHTML = '<p style="text-align: center; color: #999;">Błąd przy ładowaniu zamówień</p>';
    }
}

function displayOrders(orders) {
    const container = document.getElementById('ordersList');

    if (!orders || orders.length === 0) {
        container.innerHTML = `
            <div style="text-align: center; padding: 60px 20px;">
                <h3 style="color: #666;">Brak zamówień</h3>
                <p style="color: #999; margin-bottom: 20px;">Nie złożyłeś jeszcze żadnego zamówienia</p>
                <a href="shop.html" style="color: #667eea; text-decoration: none;">Przejdź do sklepu →</a>
            </div>
        `;
        return;
    }

    container.innerHTML = '';

    const statusLabels = {
        'pending': '⏳ Oczekujące',
        'paid': '✓ Zapłacone',
        'processing': '📦 Przetwarzane',
        'shipped': '🚚 Wysłane',
        'delivered': '✓ Dostarczone',
        'cancelled': '✕ Anulowane',
        'refunded': '↩️ Zwrócone',
    };

    orders.forEach(order => {
        const createdDate = new Date(order.created_at).toLocaleDateString('pl-PL');
        const orderCard = document.createElement('div');
        orderCard.className = 'order-card';
        orderCard.innerHTML = `
            <div class="order-card-header">
                <div class="order-info-item">
                    <span class="order-info-label">Zamówienie</span>
                    <span class="order-info-value">#${order.id.substring(0, 8).toUpperCase()}</span>
                </div>
                <div class="order-info-item">
                    <span class="order-info-label">Data</span>
                    <span class="order-info-value">${createdDate}</span>
                </div>
                <div class="order-info-item">
                    <span class="order-info-label">Razem</span>
                    <span class="order-info-value" style="color: #27ae60;">${order.total_amount.toFixed(2)} zł</span>
                </div>
                <div class="order-info-item">
                    <span class="order-info-label">Status</span>
                    <span class="order-info-value">${statusLabels[order.status]}</span>
                </div>
            </div>
        `;

        container.appendChild(orderCard);
    });
}

function logout() {
    localStorage.clear();
    window.location.href = '../index.html';
}
