const API_URL = 'http://localhost:8080';

// Check if user is already logged in
window.addEventListener('load', () => {
    const token = localStorage.getItem('access_token');
    if (token) {
        window.location.href = 'pages/shop.html';
    }
});

function toggleForms(e) {
    e.preventDefault();
    document.getElementById('loginForm').classList.toggle('active');
    document.getElementById('registerForm').classList.toggle('active');
}

async function handleLogin(e) {
    e.preventDefault();

    const email = document.getElementById('loginEmail').value;
    const password = document.getElementById('loginPassword').value;
    const errorEl = document.getElementById('loginError');

    errorEl.textContent = '';

    try {
        const response = await fetch(`${API_URL}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password }),
        });

        if (!response.ok) {
            throw new Error('Login failed');
        }

        const data = await response.json();

        // Store tokens and user info
        localStorage.setItem('access_token', data.access_token);
        localStorage.setItem('refresh_token', data.refresh_token);
        localStorage.setItem('user_id', data.user.id);
        localStorage.setItem('username', data.user.username);
        localStorage.setItem('user_email', data.user.email);

        // Redirect to shop
        window.location.href = 'pages/shop.html';
    } catch (error) {
        errorEl.textContent = 'Błędny email lub hasło';
        console.error('Login error:', error);
    }
}

async function handleRegister(e) {
    e.preventDefault();

    const email = document.getElementById('registerEmail').value;
    const username = document.getElementById('registerUsername').value;
    const fullName = document.getElementById('registerFullName').value;
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
            body: JSON.stringify({
                email,
                username,
                full_name: fullName,
                password,
            }),
        });

        if (!response.ok) {
            throw new Error('Registration failed');
        }

        const data = await response.json();

        // Store tokens and user info
        localStorage.setItem('access_token', data.access_token);
        localStorage.setItem('refresh_token', data.refresh_token);
        localStorage.setItem('user_id', data.user.id);
        localStorage.setItem('username', data.user.username);
        localStorage.setItem('user_email', data.user.email);

        // Redirect to shop
        window.location.href = 'pages/shop.html';
    } catch (error) {
        errorEl.textContent = 'Błąd przy rejestracji. Email lub username już istnieje.';
        console.error('Register error:', error);
    }
}
