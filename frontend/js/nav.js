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
  authActions.innerHTML = `
    <a href="orders.html">Moje zamówienia</a>
    <a href="dashboard.html">${dashboardLabel}</a>
    <span>Cześć, ${escapeHtml(user.username)}</span>
    <a href="#" id="logoutLink">Wyloguj</a>
  `;
  document.getElementById('logoutLink').addEventListener('click', (e) => {
    e.preventDefault();
    logout();
  });
}
