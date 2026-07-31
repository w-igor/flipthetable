function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

function getCurrentUser() {
  const raw = localStorage.getItem('user') || sessionStorage.getItem('user');
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch (err) {
    return null;
  }
}

function logout() {
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
  localStorage.removeItem('user');
  sessionStorage.removeItem('access_token');
  sessionStorage.removeItem('refresh_token');
  sessionStorage.removeItem('user');
  window.location.reload();
}

// Populates #authActions with a greeting + seller/dashboard link + logout,
// or leaves the logged-out markup (login/register links) untouched.
function updateHeaderForAuth() {
  const authActions = document.getElementById('authActions');
  if (!authActions) return;

  const user = getCurrentUser();
  if (!user) return;

  const dashboardLabel = user.is_seller ? 'Panel sprzedawcy' : 'Zostań sprzedawcą';
  const adminLink = user.is_admin
    ? `<a href="admin.html" class="nav-icon-link" title="Panel administratora" aria-label="Panel administratora">Admin</a>`
    : '';
  authActions.innerHTML = `
    <a href="favorites.html" class="nav-icon-link" title="Ulubione" aria-label="Ulubione">${Icons.heart}</a>
    <a href="messages.html" class="nav-icon-link" title="Wiadomości" aria-label="Wiadomości">${Icons.messageCircle}<span id="navUnreadBadge" class="nav-unread-badge" style="display:none;"></span></a>
    <a href="orders.html" class="nav-icon-link" title="Moje zamówienia" aria-label="Moje zamówienia">${Icons.package}</a>
    <a href="dashboard.html" class="nav-icon-link" title="${dashboardLabel}" aria-label="${dashboardLabel}">${Icons.store}</a>
    ${adminLink}
    <span class="nav-user-chip" title="${escapeHtml(user.username)}">${Icons.userRound}<span class="nav-username">${escapeHtml(user.username)}</span></span>
    <a href="#" id="logoutLink" class="nav-icon-link" title="Wyloguj" aria-label="Wyloguj">${Icons.logOut}</a>
  `;
  document.getElementById('logoutLink').addEventListener('click', (e) => {
    e.preventDefault();
    logout();
  });

  refreshUnreadBadge();
  setInterval(refreshUnreadBadge, 20000);
}

async function refreshUnreadBadge() {
  const badge = document.getElementById('navUnreadBadge');
  if (!badge || typeof authFetch !== 'function') return;
  try {
    const res = await authFetch('/messages/unread-count');
    if (!res.ok) return;
    const data = await res.json();
    if (data.count > 0) {
      badge.textContent = data.count;
      badge.style.display = 'inline-block';
    } else {
      badge.style.display = 'none';
    }
  } catch (err) {
    // odznaka po prostu nie odświeży się w tej turze
  }
}
