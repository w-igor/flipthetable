// Navigation and Header Management
// Handles user authentication status, header rendering, and navigation updates

// Safely escapes HTML special characters to prevent XSS attacks
function escapeHtml(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// Retrieves the current logged-in user from storage
function getCurrentUser() {
  const raw = localStorage.getItem('user') || sessionStorage.getItem('user');
  if (!raw) return null;
  try {
    return JSON.parse(raw);
  } catch (err) {
    return null;
  }
}

// Logs out the user by clearing all session data and disconnecting WebSocket
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

// Renders the authenticated user header with navigation links
// Safe to call repeatedly for re-renders (e.g., language changes)
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

// Initializes header for authenticated users
// Sets up real-time message badge updates via WebSocket + polling fallback
function updateHeaderForAuth() {
  if (!getCurrentUser()) return;
  renderHeaderAuthLinks();

  refreshUnreadBadge();
  // Poll for unread count every 60 seconds as fallback if WebSocket is unavailable
  setInterval(refreshUnreadBadge, 60000);
  if (typeof connectWS === 'function') connectWS();
}

// Global handler for language changes
// Re-renders the header and delegates to page-specific locale handlers
function onLocaleChange() {
  renderHeaderAuthLinks();
  if (typeof onPageLocaleChange === 'function') onPageLocaleChange();
}

// Updates the unread message badge display
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

// Fetches and updates the unread message count from the server
async function refreshUnreadBadge() {
  if (typeof authFetch !== 'function') return;
  try {
    const res = await authFetch('/messages/unread-count');
    if (!res.ok) return;
    const data = await res.json();
    setUnreadBadge(data.count);
  } catch (err) {
    // Badge will refresh on next attempt
  }
}
