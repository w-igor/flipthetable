# FlipTheTable - Etsy Clone

Monorepo dla Etsy-like marketplace z Go backendem, vanilla JS frontendem, i Neon (PostgreSQL) bazą danych.

## Struktura Projektu

```
.
├── backend/          # Go server (REST API, auth, katalog, zamówienia)
│   └── seed/         # Jednorazowy seeder danych demo (kategorie, sklep, produkty)
├── frontend/         # Vanilla JS client
├── init.sql          # Database schema (UUID, Neon)
└── README.md
```

## Zrobione ✅

### Auth
- ✅ Rejestracja i logowanie (bcrypt + JWT)
- ✅ Access token (15 min) + refresh token (24h / 30 dni z "remember me")
- ✅ Auto-refresh tokenu w froncie przy 401 (`frontend/js/api.js`)
- ✅ `GET /auth/me`

### Katalog produktów
- ✅ Przeglądanie produktów z paginacją
- ✅ Filtrowanie po kategorii, cenie, dostępności
- ✅ Wyszukiwarka (tytuł/opis)
- ✅ Sortowanie (najnowsze, cena, popularność)
- ✅ Strona szczegółów produktu

### Koszyk i checkout
- ✅ Koszyk w localStorage (dodaj/usuń/zmień ilość), szuflada w headerze
- ✅ Checkout z formularzem wysyłki
- ✅ Tworzenie zamówień w transakcji (grupowanie per sklep, blokada stanu magazynowego, walidacja ilości)
- ✅ Strona potwierdzenia zamówienia
- ✅ `GET /orders`, `GET /orders/:id` (backend gotowy, brak jeszcze frontendowej historii zamówień)

## Niezrobione jeszcze ❌

- ❌ Strona/historia zamówień użytkownika (endpoints są, brak UI)
- ❌ Statystyki zamówień (suma wydatków, liczba oczekujących)
- ❌ Dashboard sprzedawcy (CRUD produktów, zarządzanie zamówieniami, statystyki sprzedaży)
- ❌ WebSockety / powiadomienia w czasie rzeczywistym
- ❌ Recenzje produktów, ulubione (`favorites`), wiadomości (`messages`) — tabele istnieją w `init.sql`, brak API i UI
- ❌ Integracja płatności (tabela `payments` istnieje, brak logiki)
- ❌ Docker Compose / lokalny Postgres (obecnie tylko Neon)

## Wymagania

- Go 1.23+
- Python 3 (do serwowania frontendu lokalnie) lub dowolny inny static file server
- Konto Neon.tech (baza już skonfigurowana w `.env`)

## Uruchomienie lokalne

### 1. Backend
```bash
cd backend
go run .
# Server: http://localhost:8080
```
Backend czyta `.env` z katalogu głównego repo i łączy się bezpośrednio z Neon — nie potrzeba Dockera ani lokalnego Postgresa.

### 2. (Opcjonalnie) Seed danych demo
Jeśli baza jest pusta (brak kategorii/sklepów/produktów):
```bash
cd backend/seed
go run main.go
```
Tworzy demo sklep, 5 kategorii i 10 przykładowych produktów. Bezpieczne do wielokrotnego uruchamiania (nie duplikuje istniejących danych).

### 3. Frontend
```bash
cd frontend
python -m http.server 3000
```
Otwórz `http://localhost:3000/index.html` — przekierowuje do katalogu produktów (`pages/shop.html`).

## API Endpoints

### Auth
- `POST /auth/register` — rejestracja `{username, email, password, is_seller}`
- `POST /auth/login` — logowanie `{email, password, remember}`
- `POST /auth/refresh` — odświeżenie access tokena `{refresh_token}`
- `GET /auth/me` — dane zalogowanego użytkownika (wymaga `Authorization: Bearer`)

### Katalog
- `GET /categories` — lista kategorii
- `GET /listings` — lista produktów; query params: `category`, `q`, `min_price`, `max_price`, `in_stock`, `sort`, `page`, `page_size`
- `GET /listings/:id` — szczegóły produktu (zwiększa `views_count`)

### Zamówienia (wymagają auth)
- `POST /orders` — `{items: [{listing_id, quantity}], shipping_addr, note}` — tworzy zamówienie(a), grupując koszyk per sklep
- `GET /orders` — historia zamówień kupującego
- `GET /orders/:id` — szczegóły zamówienia

## Environment Variables

```
PGUSER=admin
PGPASSWORD=***
PGHOST=...neon.tech
PGPORT=5432
PGDATABASE=flipthetable
JWT_SECRET=***           # zmienić na prod!
PORT=8080
```

## Troubleshooting

### CORS błędy w frontend
Backend ma `Access-Control-Allow-Origin: *` skonfigurowane globalnie (`backend/middleware.go`).

### Token expired / 401 na chronionych endpointach
Frontend automatycznie próbuje odświeżyć token przez `/auth/refresh` (`frontend/js/api.js`, `authFetch`). Jeśli refresh token też wygasł, użytkownik jest przekierowywany do loginu.

### Database connection refused (Neon)
- Sprawdź `.env` (`PGHOST`, `PGPASSWORD` itd.)
- Neon może potrzebować chwili na "obudzenie się" po uśpieniu (cold start)

---

Built with 💜 by Inth
