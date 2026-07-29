const API_URL = window.API_URL || 'http://localhost:8080';

const state = {
  category: '',
  q: '',
  minPrice: '',
  maxPrice: '',
  inStock: false,
  sort: 'newest',
  page: 1,
  pageSize: 12,
};

function formatPrice(price, currency) {
  const value = parseFloat(price).toFixed(2);
  return `${value} ${currency}`;
}

async function loadCategories() {
  try {
    const res = await fetch(`${API_URL}/categories`);
    const categories = await res.json();
    const container = document.getElementById('categoryList');

    categories.forEach((c) => {
      const btn = document.createElement('button');
      btn.className = 'filter-category';
      btn.dataset.slug = c.slug;
      btn.textContent = c.name;
      btn.addEventListener('click', () => {
        state.category = c.slug;
        state.page = 1;
        highlightActiveCategory();
        loadListings();
      });
      container.appendChild(btn);
    });
  } catch (err) {
    console.error('Nie udało się pobrać kategorii', err);
  }
}

function highlightActiveCategory() {
  document.querySelectorAll('.filter-category').forEach((btn) => {
    btn.classList.toggle('active', btn.dataset.slug === state.category);
  });
}

function buildQuery() {
  const params = new URLSearchParams();
  if (state.category) params.set('category', state.category);
  if (state.q) params.set('q', state.q);
  if (state.minPrice) params.set('min_price', state.minPrice);
  if (state.maxPrice) params.set('max_price', state.maxPrice);
  if (state.inStock) params.set('in_stock', 'true');
  if (state.sort !== 'newest') params.set('sort', state.sort);
  params.set('page', state.page);
  params.set('page_size', state.pageSize);
  return params.toString();
}

function renderListings(data) {
  const grid = document.getElementById('listingGrid');
  const emptyState = document.getElementById('emptyState');
  const resultCount = document.getElementById('resultCount');

  grid.innerHTML = '';

  if (data.items.length === 0) {
    emptyState.style.display = 'block';
    resultCount.textContent = '';
    renderPagination(data);
    return;
  }

  emptyState.style.display = 'none';
  resultCount.textContent = `${data.total} ${data.total === 1 ? 'produkt' : 'produktów'}`;

  data.items.forEach((item) => {
    const card = document.createElement('a');
    card.className = 'listing-card';
    card.href = `listing.html?id=${item.id}`;

    const img = item.primary_photo || 'https://picsum.photos/seed/placeholder/600/600';
    const outOfStock = item.quantity === 0;

    card.innerHTML = `
      <img src="${img}" alt="${escapeHtml(item.title)}" loading="lazy" />
      <div class="listing-card-body">
        <p class="listing-card-shop">${escapeHtml(item.shop_name)}</p>
        <p class="listing-card-title">${escapeHtml(item.title)}</p>
        <p class="listing-card-price">${formatPrice(item.price, item.currency)}</p>
        ${outOfStock
          ? '<p class="listing-card-stock">Brak w magazynie</p>'
          : `<button class="listing-card-add-btn" type="button">Dodaj do koszyka</button>`}
      </div>
    `;

    if (!outOfStock) {
      const addBtn = card.querySelector('.listing-card-add-btn');
      addBtn.addEventListener('click', (e) => {
        e.preventDefault();
        e.stopPropagation();
        addToCart(item.id, 1);
        addBtn.textContent = 'Dodano ✓';
        setTimeout(() => {
          addBtn.textContent = 'Dodaj do koszyka';
        }, 1200);
      });
    }

    grid.appendChild(card);
  });

  renderPagination(data);
}

function renderPagination(data) {
  const pagination = document.getElementById('pagination');
  pagination.innerHTML = '';

  if (data.total_pages <= 1) return;

  const prevBtn = document.createElement('button');
  prevBtn.textContent = '‹';
  prevBtn.disabled = state.page <= 1;
  prevBtn.addEventListener('click', () => {
    state.page -= 1;
    loadListings();
  });
  pagination.appendChild(prevBtn);

  for (let i = 1; i <= data.total_pages; i++) {
    const btn = document.createElement('button');
    btn.textContent = i;
    if (i === state.page) btn.classList.add('active');
    btn.addEventListener('click', () => {
      state.page = i;
      loadListings();
    });
    pagination.appendChild(btn);
  }

  const nextBtn = document.createElement('button');
  nextBtn.textContent = '›';
  nextBtn.disabled = state.page >= data.total_pages;
  nextBtn.addEventListener('click', () => {
    state.page += 1;
    loadListings();
  });
  pagination.appendChild(nextBtn);
}

async function loadListings() {
  try {
    const res = await fetch(`${API_URL}/listings?${buildQuery()}`);
    const data = await res.json();
    renderListings(data);
  } catch (err) {
    console.error('Nie udało się pobrać produktów', err);
    document.getElementById('resultCount').textContent = 'Błąd ładowania produktów. Sprawdź czy backend działa.';
  }
}

function initShopPage() {
  updateHeaderForAuth();
  loadCategories();
  loadListings();

  document.getElementById('searchForm').addEventListener('submit', (e) => {
    e.preventDefault();
    state.q = document.getElementById('searchInput').value.trim();
    state.page = 1;
    loadListings();
  });

  document.getElementById('applyFilters').addEventListener('click', () => {
    state.minPrice = document.getElementById('minPrice').value;
    state.maxPrice = document.getElementById('maxPrice').value;
    state.inStock = document.getElementById('inStockOnly').checked;
    state.page = 1;
    loadListings();
  });

  document.getElementById('sortSelect').addEventListener('change', (e) => {
    state.sort = e.target.value;
    state.page = 1;
    loadListings();
  });
}

initShopPage();
