# FlipTheTable - Etsy Clone

Monorepo dla Etsy-like marketplace z Go backendem, vanilla JS frontendem, i Neon (PostgreSQL) bazą danych.

## Struktura Projektu

```
.
├── backend/          # Go server (REST API, auth)
├── frontend/         # Vanilla JS client
├── shared/           # Shared types/constants
├── docker-compose.yml
├── init.sql          # Database schema
└── README.md
```

## Stage 1: Auth & Login ✅

- ✅ User registration
- ✅ User login
- ✅ Refresh tokens
- ✅ "Remember me" (30-day sessions)
- ✅ JWT authentication
- ✅ Password hashing (bcrypt)

## Stage 2: Shop & Catalog ✅

- ✅ Product listing with filters
- ✅ Category filtering
- ✅ Search functionality
- ✅ Product details modal
- ✅ Shopping cart (add/remove/update)
- ✅ Price filtering
- ✅ Stock status display

## Stage 3: Orders & Checkout ✅

- ✅ Checkout page with shipping options
- ✅ Order creation with cart validation
- ✅ Stock management (automatic reduction)
- ✅ Order confirmation page
- ✅ Order history with filtering
- ✅ Order statistics (total spent, pending orders)
- ✅ Order detail modal
- ✅ Order status tracking
- ✅ Transaction handling (cart clear on success)

## Stage 4: WebSockets & Real-time ✅

- ✅ WebSocket server (gorilla/websocket)
- ✅ Real-time notification system
- ✅ Order creation notifications
- ✅ Order status update notifications
- ✅ Notification center UI
- ✅ Toast notifications
- ✅ Connection status indicator
- ✅ Auto-reconnect with exponential backoff
- ✅ Unread notification badge

## Stage 5: Seller Dashboard ✅

- ✅ Seller registration & profile
- ✅ Product management (CRUD)
- ✅ Product listing with edit/delete
- ✅ Inventory management
- ✅ Order management view
- ✅ Sales analytics dashboard
- ✅ Performance stats (total sales, avg rating)
- ✅ Monthly sales tracking
- ✅ Seller verification status
- ✅ Multi-tab dashboard UI

## Wymagania

- Docker & Docker Compose
- Go 1.21+
- Node.js (opcjonalnie, dla toolów)

## Setup z Neon.tech (Production Database)

### 1. Clone i setup
```bash
cd flipthetable
cp .env.example .env
```

### 2. Skonfiguruj .env dla Neon
Edytuj `.env` i dodaj dane z Neon:
```bash
PGUSER=admin
PGPASSWORD=npg_TokODPY2bS4f
PGHOST=ep-still-band-aiqm10i0-pooler.c-4.us-east-1.aws.neon.tech
PGPORT=5432
PGDATABASE=flipthetable
```

### 3. Inicjalizuj bazę danych (Neon)
1. Przejdź do [Neon Console](https://console.neon.tech)
2. Otwórz projekt → SQL Editor
3. Skopiuj zawartość `init.sql` i wykonaj zapytania
4. Czekaj na "Queries executed successfully"

### 4. Backend
```bash
cd backend
go mod download
go run main.go
# Server: http://localhost:8080 ✓
```

### 5. Frontend
```bash
# Otwórz frontend/index.html w przeglądarce
# Lub uruchom:
cd frontend
python -m http.server 3000
# Otwórz: http://localhost:3000 ✓
```

## Setup Lokalny (z Docker - opcjonalnie)
```bash
docker-compose up -d
# Uruchamia lokalny PostgreSQL zamiast Neon
```

## API Endpoints

### Auth
- `POST /auth/register` - Rejestracja
- `POST /auth/login` - Logowanie
- `POST /auth/refresh` - Odświeżenie access tokena
- `GET /auth/me` - Dane użytkownika (wymaga auth)

### Request Body (register/login)
```json
{
  "email": "user@example.com",
  "password": "password123",
  "remember": true
}
```

### Response (login/register)
```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "eyJhbG...",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "created_at": "2026-07-29T10:00:00Z"
  }
}
```

## Frontend Features

- Login/Signup UI
- Token storage (localStorage)
- Auto-refresh na token expiry
- "Remember me" (30 dni)
- Dashboard po zalogowaniu

## Następne Etapy

- [ ] Stage 2: Shop/Catalog (REST API)
- [ ] Stage 3: WebSockets (Live chat, notifications)
- [ ] Stage 4: Dashboard (Stats, history)
- [ ] Stage 5: Payment integration

## Environment Variables

```
PGUSER=admin              # DB user
PGPASSWORD=***           # DB password
PGHOST=localhost         # DB host
PGPORT=5432              # DB port
PGDATABASE=flipthetable  # DB name
JWT_SECRET=***           # JWT secret (zmienić na prod!)
PORT=8080                # Server port
```

## Production Deployment

### Verify Neon Connection
```bash
# Test connection (install psql first):
psql "postgresql://admin:npg_TokODPY2bS4f@ep-still-band-aiqm10i0-pooler.c-4.us-east-1.aws.neon.tech/flipthetable?sslmode=require"
```

### Environment Variables (production)
Upewnij się że masz w `.env`:
```
JWT_SECRET=change-this-strong-random-key
PGPASSWORD=npg_TokODPY2bS4f
```

### Deploy Backend (Render/Railway/Heroku)
1. Push to GitHub
2. Connect repository w platform'ie
3. Set env variables
4. Deploy

### Deploy Frontend (Vercel/Netlify)
1. Zmień `API_URL` na production backend
2. Deploy `frontend/` folder

## Troubleshooting

### CORS błędy w frontend
- Backend musi mieć CORS headers (już configured)
- Frontend robiący fetch powinien mieć `http://localhost:8080` (dev) lub production URL

### Token expired
- Frontend automatycznie refreshuje token
- Jeśli refresh token wygasł, user musi się zalogować ponownie

### Database connection refused (Neon)
- Sprawdź czy connection string jest poprawny
- Czekaj ~30s na cold start (Neon może być w sleep mode)
- Verify IP whitelisting w Neon settings

## Development

```bash
# Watch mode (nie wbudowany, ale możesz użyć air w Go)
# Dla frontend: live-server frontend/

# Testing
cd backend
go test ./...
```

---

Built with 💜 by Inth
