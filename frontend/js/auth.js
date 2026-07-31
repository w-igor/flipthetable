const API_URL = window.API_URL || 'http://localhost:8080';

function showBanner(type, message) {
  const banner = document.getElementById(type === 'error' ? 'errorBanner' : 'successBanner');
  const other = document.getElementById(type === 'error' ? 'successBanner' : 'errorBanner');
  other.classList.remove('show');
  banner.textContent = message;
  banner.classList.add('show');
}

function clearBanners() {
  document.getElementById('errorBanner').classList.remove('show');
  document.getElementById('successBanner').classList.remove('show');
}

function setFieldError(fieldId, message) {
  const input = document.getElementById(fieldId);
  const errorEl = document.getElementById(fieldId + 'Error');
  if (message) {
    input.classList.add('invalid');
    errorEl.textContent = message;
  } else {
    input.classList.remove('invalid');
    errorEl.textContent = '';
  }
}

function isValidEmail(email) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

function storeSession(data, remember) {
  const storage = remember ? localStorage : sessionStorage;
  storage.setItem('access_token', data.access_token);
  storage.setItem('refresh_token', data.refresh_token);
  storage.setItem('user', JSON.stringify(data.user));
}

function togglePasswordVisibility() {
  document.querySelectorAll('.auth-toggle-password').forEach((btn) => {
    btn.addEventListener('click', () => {
      const target = document.getElementById(btn.dataset.target);
      const isHidden = target.type === 'password';
      target.type = isHidden ? 'text' : 'password';
      btn.textContent = isHidden ? t('auth.hide_password') : t('auth.show_password');
    });
  });
}

function initLoginPage() {
  const form = document.getElementById('loginForm');
  const submitBtn = document.getElementById('submitBtn');

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearBanners();
    setFieldError('email', '');
    setFieldError('password', '');

    const email = document.getElementById('email').value.trim();
    const password = document.getElementById('password').value;
    const remember = document.getElementById('remember').checked;

    let hasError = false;
    if (!isValidEmail(email)) {
      setFieldError('email', t('auth.err_invalid_email'));
      hasError = true;
    }
    if (!password) {
      setFieldError('password', t('auth.err_password_required'));
      hasError = true;
    }
    if (hasError) return;

    submitBtn.disabled = true;
    submitBtn.textContent = t('auth.logging_in');

    try {
      const res = await fetch(`${API_URL}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password, remember }),
      });
      const data = await res.json();

      if (!res.ok) {
        showBanner('error', data.message || t('auth.err_invalid_credentials'));
        return;
      }

      storeSession(data, remember);
      showBanner('success', t('auth.success_login'));
      const redirect = new URLSearchParams(window.location.search).get('redirect');
      setTimeout(() => {
        window.location.href = redirect ? redirect : '../index.html';
      }, 800);
    } catch (err) {
      showBanner('error', t('auth.err_connect_retry'));
    } finally {
      submitBtn.disabled = false;
      submitBtn.textContent = t('auth.login_title');
    }
  });
}

function initRegisterPage() {
  const form = document.getElementById('registerForm');
  const submitBtn = document.getElementById('submitBtn');

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearBanners();
    ['username', 'email', 'password', 'confirmPassword'].forEach((f) => setFieldError(f, ''));

    const username = document.getElementById('username').value.trim();
    const email = document.getElementById('email').value.trim();
    const password = document.getElementById('password').value;
    const confirmPassword = document.getElementById('confirmPassword').value;
    const isSeller = document.getElementById('isSeller').checked;

    let hasError = false;
    if (username.length < 3) {
      setFieldError('username', t('auth.err_username_length'));
      hasError = true;
    }
    if (!isValidEmail(email)) {
      setFieldError('email', t('auth.err_invalid_email'));
      hasError = true;
    }
    if (password.length < 8) {
      setFieldError('password', t('auth.err_password_length'));
      hasError = true;
    }
    if (confirmPassword !== password) {
      setFieldError('confirmPassword', t('auth.err_password_mismatch'));
      hasError = true;
    }
    if (hasError) return;

    submitBtn.disabled = true;
    submitBtn.textContent = t('auth.creating_account');

    try {
      const res = await fetch(`${API_URL}/auth/register`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, email, password, is_seller: isSeller }),
      });
      const data = await res.json();

      if (!res.ok) {
        showBanner('error', data.message || t('auth.err_register_failed'));
        return;
      }

      storeSession(data, true);
      showBanner('success', t('auth.success_register'));
      setTimeout(() => {
        window.location.href = '../index.html';
      }, 800);
    } catch (err) {
      showBanner('error', t('auth.err_connect_retry'));
    } finally {
      submitBtn.disabled = false;
      submitBtn.textContent = t('auth.register_link');
    }
  });
}

function initAuthPage(page) {
  togglePasswordVisibility();
  if (page === 'login') initLoginPage();
  if (page === 'register') initRegisterPage();
}
