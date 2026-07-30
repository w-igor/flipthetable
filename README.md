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
- ✅ Strona „Moje zamówienia" (`pages/orders.html`) — historia zamówień kupującego z pozycjami i statusem

### Sprzedawcy
- ✅ Zakładanie sklepu (dowolne konto może „zostać sprzedawcą" zakładając sklep — `POST /shops`)
- ✅ Publiczny profil sklepu (`pages/shop-profile.html`) — baner, avatar, opis, siatka ofert
- ✅ Panel sprzedawcy (`pages/dashboard.html`): edycja profilu sklepu, CRUD ofert (dodaj/edytuj/wyłącz), zarządzanie zamówieniami (zmiana statusu), statystyki (liczba zamówień, przychód, aktywne oferty)

### Opinie
- ✅ Kupujący może ocenić zakupioną pozycję (gwiazdki 1-5 + komentarz) ze strony „Moje zamówienia" lub ze strony produktu, gdy zamówienie jest opłacone/w realizacji/wysłane/dostarczone
- ✅ `avg_rating` produktu przeliczane automatycznie po dodaniu opinii

### Płatności
- ✅ Symulowana bramka płatnicza — przy checkoucie tworzone jest zamówienie (`status = pending`) razem z rekordem w `payments` (`status = pending`)
- ✅ `POST /orders/:id/pay` — waliduje dane karty (Luhn, CVC, data ważności), symuluje autoryzację i ustawia `payments.status` (`completed`/`failed`) oraz `orders.status = paid` po sukcesie
- ✅ Karta testowa `4000000000000002` zawsze kończy się odrzuceniem (jak testowe karty Stripe) — reszta prawidłowych numerów (Luhn) jest akceptowana
- ✅ Możliwość ponowienia płatności ze strony „Moje zamówienia", jeśli pierwsza próba się nie powiedzie

### Ulubione i wiadomości
- ✅ Ulubione produkty (serduszko w katalogu, strona `pages/favorites.html`)
- ✅ Wiadomości kupujący–sprzedawca, licznik nieprzeczytanych w headerze (polling)

### Zdjęcia
- ✅ Lokalny upload zdjęć ofert i logo/baneru sklepu (`POST /uploads`)

## Niezrobione jeszcze ❌

- ❌ WebSockety / prawdziwe powiadomienia push (obecnie tylko polling dla licznika nieprzeczytanych wiadomości)
- ❌ Prawdziwy dostawca płatności (Stripe/Przelewy24 itp.) — obecnie symulacja bez integracji zewnętrznej
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
- `GET /listings` — lista produktów; query params: `category`, `shop_id`, `q`, `min_price`, `max_price`, `in_stock`, `sort`, `page`, `page_size`
- `GET /listings/:id` — szczegóły produktu (zwiększa `views_count`)

### Zamówienia (wymagają auth)
- `POST /orders` — `{items: [{listing_id, quantity}], shipping_addr, note}` — tworzy zamówienie(a), grupując koszyk per sklep
- `GET /orders` — historia zamówień kupującego (z pozycjami i flagą `reviewed` per pozycja)
- `GET /orders/:id` — szczegóły zamówienia
- `POST /orders/:id/pay` — `{cardholder_name, card_number, exp_month, exp_year, cvc}` — symulowana płatność za zamówienie (kupujący, zamówienie musi być `pending`)

### Sklepy
- `POST /shops` — zakłada sklep dla zalogowanego użytkownika (ustawia `is_seller = true`), wymaga auth
- `GET /shops/me` — dane własnego sklepu (auth)
- `PUT /shops/me` — edycja własnego sklepu (auth)
- `GET /shops/:id` — publiczny profil sklepu (nazwa, opis, baner, avatar, liczba ofert)

### Oferty sprzedawcy (wymagają auth + własnego sklepu)
- `POST /listings` — dodanie oferty do własnego sklepu
- `PUT /listings/:id` — edycja własnej oferty
- `DELETE /listings/:id` — dezaktywacja oferty (soft-delete, `is_active = false`)
- `GET /seller/listings` — wszystkie własne oferty (także nieaktywne)
- `GET /seller/stats` — statystyki sklepu (zamówienia, przychód, aktywne/wszystkie oferty)
- `GET /seller/orders` — zamówienia złożone we własnym sklepie
- `PUT /seller/orders/:id/status` — zmiana statusu zamówienia

### Opinie
- `POST /reviews` — `{order_item_id, rating, comment}` — ocena zakupionej pozycji (auth, wymaga statusu zamówienia `paid`/`processing`/`shipped`/`delivered`)
- `GET /listings/:id/reviews` — lista opinii dla produktu

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
