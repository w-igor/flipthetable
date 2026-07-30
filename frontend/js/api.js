const API_URL = window.API_URL || 'http://localhost:8080';

function getAccessToken() {
  return localStorage.getItem('access_token') || sessionStorage.getItem('access_token');
}

function getRefreshToken() {
  return localStorage.getItem('refresh_token') || sessionStorage.getItem('refresh_token');
}

function getActiveStorage() {
  return localStorage.getItem('access_token') ? localStorage : sessionStorage;
}

function clearSession() {
  [localStorage, sessionStorage].forEach((storage) => {
    storage.removeItem('access_token');
    storage.removeItem('refresh_token');
    storage.removeItem('user');
  });
}

async function refreshAccessToken() {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return null;

  try {
    const res = await fetch(`${API_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    if (!res.ok) return null;
    const data = await res.json();
    getActiveStorage().setItem('access_token', data.access_token);
    return data.access_token;
  } catch (err) {
    return null;
  }
}

// Wraps fetch for authenticated endpoints: on 401, tries a token refresh
// once and retries; if that also fails, clears the session and bounces
// to login so the user isn't left staring at a silent failure.
async function authFetch(path, options = {}) {
  const doFetch = (token) =>
    fetch(`${API_URL}${path}`, {
      ...options,
      headers: {
        ...(options.headers || {}),
        Authorization: `Bearer ${token}`,
      },
    });

  let token = getAccessToken();
  if (!token) {
    window.location.href = `login.html?redirect=${encodeURIComponent(window.location.pathname.split('/').pop() + window.location.search)}`;
    return Promise.reject(new Error('Brak sesji'));
  }

  let res = await doFetch(token);

  if (res.status === 401) {
    const newToken = await refreshAccessToken();
    if (newToken) {
      res = await doFetch(newToken);
    } else {
      clearSession();
      window.location.href = `login.html?redirect=${encodeURIComponent(window.location.pathname.split('/').pop() + window.location.search)}`;
      return Promise.reject(new Error('Sesja wygasła'));
    }
  }

  return res;
}

// Uploads a single image file and resolves to its public URL, or null on failure.
async function uploadPhoto(file) {
  const formData = new FormData();
  formData.append('photo', file);
  const res = await authFetch('/uploads', { method: 'POST', body: formData });
  if (!res.ok) return null;
  const data = await res.json();
  return data.url;
}
