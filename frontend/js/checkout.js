const API_URL = 'http://localhost:8080';

let cart = [];
let allListings = [];

// Initialize
window.addEventListener('load', () => {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = '../index.html';
        return;
    }

    loadCart();
});

function loadCart() {
    const savedCart = localStorage.getItem('cart');
    if (savedCart) {
        cart = JSON.parse(savedCart);
    }

    if (cart.length === 0) {
        document.body.innerHTML = '<div style="text-align: center; padding: 50px;"><h2>Koszyk jest pusty</h2><a href="shop.html">← Wróć do sklepu</a></div>';
        return;
    }

    displayOrderSummary();
}

async function displayOrderSummary() {
    try {
        // Load all listings
        const response = await fetch(`${API_URL}/api/listings`);
        allListings = await response.json() || [];

        const container = document.getElementById('orderItems');
        container.innerHTML = '';

        let total = 0;

        cart.forEach(item => {
            const listing = allListings.find(l => l.id === item.listing_id);
            if (listing) {
                total += listing.price * item.quantity;

                const orderItem = document.createElement('div');
                orderItem.className = 'order-item';
                orderItem.innerHTML = `
                    <img src="${listing.photos?.[0]?.url || 'https://via.placeholder.com/200'}" alt="${listing.title}" class="order-item-image" />
                    <div class="order-item-details">
                        <div class="order-item-name">${listing.title}</div>
                        <div class="order-item-qty">Ilość: ${item.quantity}</div>
                        <div class="order-item-price">${(listing.price * item.quantity).toFixed(2)} zł</div>
                    </div>
                `;
                container.appendChild(orderItem);
            }
        });

        document.getElementById('totalPrice').textContent = total.toFixed(2) + ' zł';
    } catch (error) {
        console.error('Failed to load cart:', error);
    }
}

async function handleCheckout(event) {
    event.preventDefault();

    const fullName = document.getElementById('fullName').value;
    const address = document.getElementById('address').value;
    const city = document.getElementById('city').value;
    const postalCode = document.getElementById('postalCode').value;
    const country = document.getElementById('country').value;
    const phone = document.getElementById('phone').value;

    if (!fullName || !address || !city || !postalCode || !country || !phone) {
        alert('Uzupełnij wszystkie pola');
        return;
    }

    // Get shop ID from first listing
    const firstListing = allListings.find(l => l.id === cart[0].listing_id);
    if (!firstListing) {
        alert('Błąd przy ładowaniu produktu');
        return;
    }

    const shippingAddr = {
        full_name: fullName,
        address: address,
        city: city,
        postal_code: postalCode,
        country: country,
        phone: phone,
    };

    const orderItems = cart.map(item => ({
        listing_id: item.listing_id,
        quantity: item.quantity,
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
                shop_id: firstListing.shop_id,
                items: orderItems,
                shipping_addr: shippingAddr,
                note: '',
            }),
        });

        if (response.ok) {
            const order = await response.json();

            // Clear cart
            localStorage.removeItem('cart');

            // Redirect to order confirmation
            window.location.href = `order-confirmation.html?id=${order.id}`;
        } else {
            const error = await response.json();
            alert('Błąd przy tworzeniu zamówienia: ' + (error.message || 'Nieznany błąd'));
        }
    } catch (error) {
        console.error('Failed to create order:', error);
        alert('Błąd przy tworzeniu zamówienia');
    }
}
