const reviewableStatuses = ['paid', 'processing', 'shipped', 'delivered'];

const orderStatusLabels = {
  pending: 'Oczekujące',
  paid: 'Opłacone',
  processing: 'W realizacji',
  shipped: 'Wysłane',
  delivered: 'Dostarczone',
  cancelled: 'Anulowane',
  refunded: 'Zwrócone',
};

function requireLogin() {
  const user = getCurrentUser();
  if (!user) {
    window.location.href = 'login.html?redirect=orders.html';
    return null;
  }
  return user;
}

function renderOrderItem(order, item) {
  const img = item.photo_url || 'https://picsum.photos/seed/placeholder/100/100';
  const canReview = reviewableStatuses.includes(order.status);

  let actionHtml;
  if (item.reviewed) {
    actionHtml = '<p class="order-item-reviewed">Oceniono ✓</p>';
  } else if (canReview) {
    actionHtml = `
      <form class="order-review-form" data-item-id="${item.id}">
        <select class="order-review-rating">
          <option value="5">5 - świetnie</option>
          <option value="4">4 - dobrze</option>
          <option value="3">3 - średnio</option>
          <option value="2">2 - słabo</option>
          <option value="1">1 - źle</option>
        </select>
        <textarea class="order-review-comment" rows="2" placeholder="Komentarz (opcjonalnie)"></textarea>
        <button type="submit">Dodaj opinię</button>
        <p class="order-review-error"></p>
      </form>
    `;
  } else {
    actionHtml = '<p class="order-item-wait">Poczekaj, aż sprzedawca zaktualizuje status zamówienia.</p>';
  }

  return `
    <div class="order-item-row">
      <img src="${img}" alt="${escapeHtml(item.title_snapshot)}" />
      <div class="order-item-info">
        <a class="order-item-title" href="listing.html?id=${item.listing_id}">${escapeHtml(item.title_snapshot)}</a>
        <p class="order-item-meta">${item.quantity} × ${parseFloat(item.unit_price).toFixed(2)} PLN</p>
        ${actionHtml}
      </div>
    </div>
  `;
}

function renderOrders(orders) {
  const container = document.getElementById('ordersList');
  if (orders.length === 0) {
    container.innerHTML = '<p class="orders-empty">Nie masz jeszcze żadnych zamówień.</p>';
    return;
  }

  container.innerHTML = orders
    .map(
      (order) => `
    <div class="order-card">
      <div class="order-card-header">
        <div>
          <p class="order-card-shop">${escapeHtml(order.shop_name)}</p>
          <p class="order-card-meta">Zamówienie #${order.id.slice(0, 8)} · ${new Date(order.created_at).toLocaleDateString('pl-PL')}</p>
        </div>
        <div class="order-card-status">
          <span class="order-status-badge order-status-${order.status}">${orderStatusLabels[order.status] || order.status}</span>
          <p class="order-card-total">${parseFloat(order.total_amount).toFixed(2)} ${order.currency}</p>
        </div>
      </div>
      <div class="order-card-items">
        ${(order.items || []).map((item) => renderOrderItem(order, item)).join('')}
      </div>
    </div>
  `
    )
    .join('');

  container.querySelectorAll('.order-review-form').forEach((form) => {
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      const itemId = form.dataset.itemId;
      const rating = parseInt(form.querySelector('.order-review-rating').value, 10);
      const comment = form.querySelector('.order-review-comment').value.trim();
      const errorEl = form.querySelector('.order-review-error');
      const submitBtn = form.querySelector('button');
      errorEl.textContent = '';
      submitBtn.disabled = true;

      try {
        const res = await authFetch('/reviews', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ order_item_id: itemId, rating, comment }),
        });
        const data = await res.json();
        if (!res.ok) {
          errorEl.textContent = data.message || 'Nie udało się dodać opinii.';
          submitBtn.disabled = false;
          return;
        }
        form.outerHTML = '<p class="order-item-reviewed">Oceniono ✓</p>';
      } catch (err) {
        errorEl.textContent = 'Nie udało się połączyć z serwerem.';
        submitBtn.disabled = false;
      }
    });
  });
}

async function loadOrders() {
  const container = document.getElementById('ordersList');
  container.innerHTML = '<p class="orders-empty">Ładowanie...</p>';
  try {
    const res = await authFetch('/orders');
    const orders = await res.json();
    if (!res.ok) {
      container.innerHTML = '<p class="orders-empty">Nie udało się pobrać zamówień.</p>';
      return;
    }
    renderOrders(orders);
  } catch (err) {
    container.innerHTML = '<p class="orders-empty">Błąd ładowania zamówień.</p>';
  }
}

function init() {
  if (!requireLogin()) return;
  updateHeaderForAuth();
  loadOrders();
}

init();
