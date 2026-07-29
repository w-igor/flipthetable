const API_URL = 'http://localhost:8080';

let cartItems = [];
const shippingCosts = {
    standard: 9.99,
    express: 24.99,
    paczkomat: 4.99
};

// Initialize
window.addEventListener('load', () => {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = '../index.html';
        return;
    }

    loadCart();
});

async function loadCart() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/cart`, {
            headers: { 'Authorization': `Bearer ${token}` },
        });

        cartItems = await response.json() || [];

        if (cartItems.length === 0) {
            document.body.innerHTML = '<div style="text-align: center; padding: 50px;"><h2>Koszyk jest pusty</h2><a href="shop.html">← Wróć do sklepu</a></div>';
            return;
        }

        displayOrderSummary();
    } catch (error) {
        console.error('Failed to load cart:', error);
        alert('Błąd podczas ładowania koszyka');
    }
}

function displayOrderSummary() {
    const container = document.getElementById('orderItems');
    container.innerHTML = '';

    let subtotal = 0;

    cartItems.forEach(item => {
        subtotal += item.product.price * item.quantity;

        const orderItem = document.createElement('div');
        orderItem.className = 'order-item';
        orderItem.innerHTML = `
            <img src="${item.product.image_url}" alt="${item.product.name}" class="order-item-image" />
            <div class="order-item-details">
                <div class="order-item-name">${item.product.name}</div>
                <div class="order-item-qty">Ilość: ${item.quantity}</div>
                <div class="order-item-price">${(item.product.price * item.quantity).toFixed(2)} zł</div>
            </div>
        `;
        container.appendChild(orderItem);
    });

    updateTotals();
}

function updateShipping() {
    updateTotals();
}

function updateTotals() {
    const shippingMethod = document.querySelector('input[name="shipping"]:checked').value;
    const shippingCost = shippingCosts[shippingMethod];

    let subtotal = 0;
    cartItems.forEach(item => {
        subtotal += item.product.price * item.quantity;
    });

    const tax = Math.round(subtotal * 0.23 * 100) / 100;
    const total = subtotal + shippingCost + tax;

    document.getElementById('subtotal').textContent = subtotal.toFixed(2) + ' zł';
    document.getElementById('shipping').textContent = shippingCost.toFixed(2) + ' zł';
    document.getElementById('tax').textContent = tax.toFixed(2) + ' zł';
    document.getElementById('totalPrice').textContent = total.toFixed(2) + ' zł';
}

document.getElementById('checkoutForm').addEventListener('submit', async (e) => {
    e.preventDefault();

    const fullName = document.getElementById('fullName').value;
    const address = document.getElementById('address').value;
    const city = document.getElementById('city').value;
    const postalCode = document.getElementById('postalCode').value;
    const country = document.getElementById('country').value;
    const phone = document.getElementById('phone').value;
    const shippingMethod = document.querySelector('input[name="shipping"]:checked').value;
    const paymentMethod = document.querySelector('input[name="payment"]:checked').value;

    if (!fullName || !address || !city || !postalCode || !country || !phone) {
        alert('Uzupełnij wszystkie pola');
        return;
    }

    // Prepare order items from cart
    const orderItems = cartItems.map(item => ({
        product_id: item.product_id,
        quantity: item.quantity
    }));

    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/api/orders`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`,
            },
            body: JSON.stringify({
                items: orderItems,
                shipping_method: shippingMethod,
                payment_method: paymentMethod,
                delivery_address: {
                    full_name: fullName,
                    address: address,
                    city: city,
                    postal_code: postalCode,
                    country: country,
                    phone: phone,
                },
            }),
        });

        if (response.ok) {
            const order = await response.json();

            // Store order info
            localStorage.setItem('lastOrderId', order.id);

            // Redirect to success page
            window.location.href = `order-confirmation.html?id=${order.id}`;
        } else {
            const error = await response.json();
            alert('Błąd przy tworzeniu zamówienia: ' + (error.message || 'Nieznany błąd'));
        }
    } catch (error) {
        console.error('Failed to create order:', error);
        alert('Błąd przy tworzeniu zamówienia');
    }
});
