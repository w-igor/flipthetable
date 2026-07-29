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

## Wymagania

- Docker & Docker Compose
- Go 1.21+
- Node.js (opcjonalnie, dla toolów)

## Setup Lokalny

### 1. Clone i setup
```bash
cd flipthetable
cp .env.example .env
```

### 2. Database (Neon cloud)
1. Stwórz projekt w [Neon](https://neon.tech)
2. Przejdź do SQL Editor
3. Skopiuj zawartość `init.sql` i wykonaj
4. Zaktualizuj `PGPASSWORD` w `admin_passwd_neon.env` danymi z Neon

### 3. Tymczasowo: Database (Docker)
```bash
docker-compose up -d
# Czeka 10 sekund na health check
```

### 4. Backend
```bash
cd backend
go mod download
go run main.go
# Server: http://localhost:8080
```

### 5. Frontend
```bash
# Otwórz frontend/index.html w przeglądarce
# Lub uruchom prosty serwer:
cd frontend
python -m http.server 3000
# Otwórz: http://localhost:3000
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

## Troubleshooting

### CORS błędy w frontend
- Backend musi mieć CORS headers (już configured)
- Frontend robiący fetch powinien mieć `http://localhost:8080`

### Token expired
- Frontend automatycznie refreshuje token
- Jeśli refresh token wygasł, user musi się zalogować ponownie

### Database connection refused
- Upewnij się że Docker kontener jest running
- Sprawdź `docker-compose logs postgres`

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
