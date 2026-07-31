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
  if (typeof disconnectWS === 'function') disconnectWS();
  localStorage.removeItem('access_token');
  localStorage.removeItem('refresh_token');
  localStorage.removeItem('user');
  sessionStorage.removeItem('access_token');
  sessionStorage.removeItem('refresh_token');
  sessionStorage.removeItem('user');
  window.location.reload();
}

// Renders the logged-in header markup. Safe to call repeatedly (e.g. on a
// language change) since it doesn't touch polling/WS setup.
function renderHeaderAuthLinks() {
  const authActions = document.getElementById('authActions');
  if (!authActions) return;

  const user = getCurrentUser();
  if (!user) return;

  const dashboardLabel = user.is_seller ? t('nav.seller_panel') : t('nav.become_seller');
  const adminLabel = t('nav.admin_panel');
  const adminLink = user.is_admin
    ? `<a href="admin.html" class="nav-icon-link" title="${adminLabel}" aria-label="${adminLabel}">Admin</a>`
    : '';
  const unreadCount = document.getElementById('navUnreadBadge');
  const previousCount = unreadCount && unreadCount.style.display !== 'none' ? unreadCount.textContent : null;
  authActions.innerHTML = `
    <a href="favorites.html" class="nav-icon-link" title="${t('nav.favorites')}" aria-label="${t('nav.favorites')}">${Icons.heart}</a>
    <a href="messages.html" class="nav-icon-link" title="${t('nav.messages')}" aria-label="${t('nav.messages')}">${Icons.messageCircle}<span id="navUnreadBadge" class="nav-unread-badge" style="display:none;"></span></a>
    <a href="orders.html" class="nav-icon-link" title="${t('nav.orders')}" aria-label="${t('nav.orders')}">${Icons.package}</a>
    <a href="dashboard.html" class="nav-icon-link" title="${dashboardLabel}" aria-label="${dashboardLabel}">${Icons.store}</a>
    ${adminLink}
    <span class="nav-user-chip" title="${escapeHtml(user.username)}">${Icons.userRound}<span class="nav-username">${escapeHtml(user.username)}</span></span>
    <a href="#" id="logoutLink" class="nav-icon-link" title="${t('nav.logout')}" aria-label="${t('nav.logout')}">${Icons.logOut}</a>
  `;
  document.getElementById('logoutLink').addEventListener('click', (e) => {
    e.preventDefault();
    logout();
  });
  if (previousCount !== null) setUnreadBadge(parseInt(previousCount, 10));
}

// Populates #authActions with a greeting + seller/dashboard link + logout,
// or leaves the logged-out markup (login/register links) untouched.
function updateHeaderForAuth() {
  if (!getCurrentUser()) return;
  renderHeaderAuthLinks();

  refreshUnreadBadge();
  // Live updates arrive over the WebSocket; this is just the slow-poll
  // fallback in case the socket is down for an extended stretch.
  setInterval(refreshUnreadBadge, 60000);
  if (typeof connectWS === 'function') connectWS();
}

// Re-render the header (and any page-specific dynamic content) when the
// user switches language via the header <select>. Defined here so ws.js's
// event dispatch and pages without a logged-in header don't need to guard.
function onLocaleChange() {
  renderHeaderAuthLinks();
  if (typeof onPageLocaleChange === 'function') onPageLocaleChange();
}

function setUnreadBadge(count) {
  const badge = document.getElementById('navUnreadBadge');
  if (!badge) return;
  if (count > 0) {
    badge.textContent = count;
    badge.style.display = 'inline-block';
  } else {
    badge.style.display = 'none';
  }
}

async function refreshUnreadBadge() {
  if (typeof authFetch !== 'function') return;
  try {
    const res = await authFetch('/messages/unread-count');
    if (!res.ok) return;
    const data = await res.json();
    setUnreadBadge(data.count);
  } catch (err) {
    // odznaka po prostu nie odświeży się w tej turze
  }
}
