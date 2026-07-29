const API_URL = 'http://localhost:8080';

const loginForm = document.getElementById('loginForm');
const registerForm = document.getElementById('registerForm');
const dashboard = document.getElementById('dashboard');

// Check if user is already logged in
window.addEventListener('load', () => {
    const token = localStorage.getItem('access_token');
    if (token) {
        fetchAndShowUser();
    }
});

function toggleForms(e) {
    e.preventDefault();
    loginForm.classList.toggle('active');
    registerForm.classList.toggle('active');
}

async function handleLogin(e) {
    e.preventDefault();

    const email = document.getElementById('loginEmail').value;
    const password = document.getElementById('loginPassword').value;
    const remember = document.getElementById('rememberMe').checked;
    const errorEl = document.getElementById('loginError');

    errorEl.textContent = '';

    try {
        const response = await fetch(`${API_URL}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password, remember }),
        });

        if (!response.ok) {
            throw new Error('Login failed');
        }

        const data = await response.json();

        // Store tokens
        localStorage.setItem('access_token', data.access_token);
        localStorage.setItem('refresh_token', data.refresh_token);
        localStorage.setItem('user_email', data.user.email);

        // Show dashboard
        showDashboard(data.user);
    } catch (error) {
        errorEl.textContent = 'Błędny email lub hasło';
        console.error('Login error:', error);
    }
}

async function handleRegister(e) {
    e.preventDefault();

    const email = document.getElementById('registerEmail').value;
    const password = document.getElementById('registerPassword').value;
    const password2 = document.getElementById('registerPassword2').value;
    const errorEl = document.getElementById('registerError');

    errorEl.textContent = '';

    if (password !== password2) {
        errorEl.textContent = 'Hasła nie są identyczne';
        return;
    }

    try {
        const response = await fetch(`${API_URL}/auth/register`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password, remember: false }),
        });

        if (!response.ok) {
            throw new Error('Registration failed');
        }

        const data = await response.json();

        // Store tokens
        localStorage.setItem('access_token', data.access_token);
        localStorage.setItem('refresh_token', data.refresh_token);
        localStorage.setItem('user_email', data.user.email);

        // Show dashboard
        showDashboard(data.user);
    } catch (error) {
        errorEl.textContent = 'Email już istnieje lub błąd przy rejestracji';
        console.error('Register error:', error);
    }
}

function showDashboard(user) {
    loginForm.classList.remove('active');
    registerForm.classList.remove('active');
    dashboard.classList.add('active');

    document.getElementById('userEmail').textContent = user.email;
    document.getElementById('userData').textContent = JSON.stringify(user, null, 2);

    // Redirect to shop after 1 second
    setTimeout(() => {
        window.location.href = 'pages/shop.html';
    }, 1000);
}

async function fetchAndShowUser() {
    try {
        const token = localStorage.getItem('access_token');
        const response = await fetch(`${API_URL}/auth/me`, {
            headers: {
                'Authorization': `Bearer ${token}`,
            },
        });

        if (response.ok) {
            const user = await response.json();
            showDashboard(user);
        } else {
            // Try refresh token
            await refreshToken();
        }
    } catch (error) {
        console.error('Fetch user error:', error);
        logout();
    }
}

async function refreshToken() {
    try {
        const refreshToken = localStorage.getItem('refresh_token');
        const response = await fetch(`${API_URL}/auth/refresh`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ refresh_token: refreshToken }),
        });

        if (response.ok) {
            const data = await response.json();
            localStorage.setItem('access_token', data.access_token);
            localStorage.setItem('refresh_token', data.refresh_token);
            fetchAndShowUser();
        } else {
            logout();
        }
    } catch (error) {
        console.error('Token refresh error:', error);
        logout();
    }
}

function logout() {
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user_email');

    loginForm.classList.add('active');
    registerForm.classList.remove('active');
    dashboard.classList.remove('active');

    document.getElementById('loginEmail').value = '';
    document.getElementById('loginPassword').value = '';
    document.getElementById('rememberMe').checked = false;
}
