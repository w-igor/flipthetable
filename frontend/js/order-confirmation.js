const API_URL = 'http://localhost:8080';

// Initialize
window.addEventListener('load', () => {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = '../index.html';
        return;
    }

    const params = new URLSearchParams(window.location.search);
    const orderId = params.get('id');

    if (!orderId) {
        window.location.href = 'shop.html';
        return;
    }

    loadOrder(orderId);
});

async function loadOrder(orderId) {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/orders/${orderId}`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        if (!response.ok) throw new Error('Order not found');

        const order = await response.json();
        displayOrderConfirmation(order);
    } catch (error) {
        console.error('Failed to load order:', error);
        alert('Błąd przy ładowaniu zamówienia');
        window.location.href = 'shop.html';
    }
}

function displayOrderConfirmation(order) {
    document.getElementById('orderNumber').textContent = order.id;

    let total = 0;
    let itemsHTML = '';

    if (order.items && order.items.length > 0) {
        order.items.forEach(item => {
            itemsHTML += `
                <div class="order-item-summary">
                    ${item.product.name} × ${item.quantity} = ${(item.price_at_purchase * item.quantity).toFixed(2)} zł
                </div>
            `;
            total += item.price_at_purchase * item.quantity;
        });
    }

    const orderInfoHTML = `
        <div class="info-row">
            <span class="info-label">Data zamówienia:</span>
            <span class="info-value">${new Date(order.created_at).toLocaleDateString('pl-PL')}</span>
        </div>
        <div class="info-row">
            <span class="info-label">Status:</span>
            <span class="info-value">${getStatusLabel(order.status)}</span>
        </div>
        <div class="info-row">
            <span class="info-label">Produkty:</span>
        </div>
        <div class="order-items-summary">
            ${itemsHTML}
        </div>
        <div class="info-row">
            <span class="info-label">Razem:</span>
            <span class="info-value">${order.total_price.toFixed(2)} zł</span>
        </div>
    `;

    document.getElementById('orderInfo').innerHTML = orderInfoHTML;
}

function getStatusLabel(status) {
    const labels = {
        'pending': '⏳ Oczekujące',
        'confirmed': '✓ Potwierdzone',
        'shipped': '📦 Wysłane',
        'delivered': '✓ Dostarczone',
        'cancelled': '✕ Anulowane'
    };
    return labels[status] || status;
}

function goToOrders() {
    window.location.href = 'orders.html';
}

function continueShopping() {
    window.location.href = 'shop.html';
}
