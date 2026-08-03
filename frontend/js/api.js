// API Configuration and Authentication Management
// Handles token refresh, session management, and authenticated requests

const API_URL = window.API_URL || 'http://localhost:8080';

// Retrieves the access token from localStorage or sessionStorage
function getAccessToken() {
  return localStorage.getItem('access_token') || sessionStorage.getItem('access_token');
}

// Retrieves the refresh token from localStorage or sessionStorage
function getRefreshToken() {
  return localStorage.getItem('refresh_token') || sessionStorage.getItem('refresh_token');
}

// Determines which storage mechanism (localStorage/sessionStorage) is currently in use
function getActiveStorage() {
  return localStorage.getItem('access_token') ? localStorage : sessionStorage;
}

// Clears all authentication data from both storage mechanisms
function clearSession() {
  [localStorage, sessionStorage].forEach((storage) => {
    storage.removeItem('access_token');
    storage.removeItem('refresh_token');
    storage.removeItem('user');
  });
}

// Requests a new access token using the refresh token
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

// Makes authenticated API requests with automatic token refresh on 401
// Redirects to login if no token is available or refresh fails
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

  // If token expired, attempt refresh and retry once
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

// Uploads a single image file to the server and returns its public URL
async function uploadPhoto(file) {
  const formData = new FormData();
  formData.append('photo', file);
  const res = await authFetch('/uploads', { method: 'POST', body: formData });
  if (!res.ok) return null;
  const data = await res.json();
  return data.url;
}
