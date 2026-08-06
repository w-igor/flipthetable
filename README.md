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
- ✅ Panel sprzedawcy (`pages/dashboard.html`): edycja profilu sklepu, CRUD ofert (dodaj/edytuj/wyłącz/usuń), filtrowanie i sortowanie własnych ofert (status, kategoria, wyszukiwanie, cena/ilość/tytuł), zarządzanie zamówieniami (zmiana statusu), statystyki (liczba zamówień, przychód, aktywne oferty)
- ✅ Usuwanie oferty — prawdziwe skasowanie (nie tylko dezaktywacja); jeśli oferta ma już historię zamówień, backend blokuje usunięcie i podpowiada wyłączenie zamiast tego
- ✅ Warianty ofert — sprzedawca definiuje do 2 własnych typów wariacji (np. Kolor, Rozmiar), każdy z dowolnymi wartościami; każda kombinacja ma własną ilość na stanie i opcjonalną własną cenę (puste = cena bazowa). Kupujący wybiera wariant na stronie produktu; stan magazynowy jest blokowany per-kombinacja przy składaniu zamówienia, a etykieta wariantu jest zapisywana w historii zamówienia nawet po usunięciu wariantu przez sprzedawcę.
- ✅ Profile wysyłki — sprzedawca może mieć kilka profili (nazwa, cena, zakres dni dostawy) i przypisać jeden do każdej oferty. Przy zamówieniu z wielu ofert jednego sklepu naliczana jest jedna opłata za wysyłkę — najdroższy z profili użytych w zamówieniu (nie suma), doliczana do `total_amount` zamówienia.

### Opinie
- ✅ Kupujący może ocenić zakupioną pozycję (gwiazdki 1-5 + komentarz) ze strony „Moje zamówienia" lub ze strony produktu, gdy zamówienie jest opłacone/w realizacji/wysłane/dostarczone
- ✅ `avg_rating` produktu przeliczane automatycznie po dodaniu opinii

### Płatności
- ✅ Prawdziwa integracja ze Stripe (Stripe Checkout) — przy checkoucie tworzone jest zamówienie (`status = pending`) razem z rekordem w `payments` (`status = pending`)
- ✅ `POST /orders/checkout-session` — tworzy Stripe Checkout Session (pozycje z `order_items` + koszt wysyłki) i zwraca URL, na który przekierowywany jest kupujący; numer karty nigdy nie trafia do naszego serwera
- ✅ `POST /webhooks/stripe` — nasłuchuje zdarzeń Stripe (`checkout.session.completed`, `checkout.session.async_payment_failed`, `checkout.session.expired`) z weryfikacją podpisu (`STRIPE_WEBHOOK_SECRET`) i na tej podstawie ustawia `payments.status` oraz `orders.status = paid`
- ✅ Możliwość ponowienia płatności ze strony „Moje zamówienia" (nowa sesja Stripe Checkout), jeśli pierwsza próba się nie powiedzie lub została anulowana

### Ulubione i wiadomości
- ✅ Ulubione produkty (serduszko w katalogu, strona `pages/favorites.html`)
- ✅ Wiadomości kupujący–sprzedawca, licznik nieprzeczytanych w headerze
- ✅ Powiadomienia push na żywo przez WebSockety (`GET /ws`, `backend/ws.go`) — nowe wiadomości i licznik nieprzeczytanych aktualizują się bez odpytywania serwera; polling co 60s zostaje tylko jako fallback na wypadek zerwanego połączenia

### Zdjęcia
- ✅ Lokalny upload zdjęć ofert i logo/baneru sklepu (`POST /uploads`)
- ✅ Do 8 zdjęć na ofertę — miniatury z podglądem i usuwaniem w panelu sprzedawcy, pierwsze zdjęcie jest okładką; na stronie produktu galeria z klikalnymi miniaturami

### Wielojęzyczność
- ✅ Frontend dostępny w 3 językach — polski, angielski, niemiecki (`frontend/js/i18n.js`) — przełącznik w nagłówku każdej strony, wybór zapamiętywany w `localStorage`
- ✅ Wszystkie statyczne teksty i treści budowane dynamicznie w JS przechodzą przez `t()`/`tn()` (z obsługą polskiej odmiany liczebników); zmiana języka odświeża też już załadowane dane (listy ofert, zamówień, wiadomości itd.) bez przeładowania strony
- ❌ Komunikaty błędów zwracane przez backend (Go) są na razie tylko po polsku — wymaga osobnego refaktoru na kody błędów

### Panel administracyjny
- ✅ Rola `is_admin` na koncie + middleware `requireAdmin` (`backend/middleware.go`)
- ✅ Panel (`pages/admin.html`) w tym samym froncie, chroniony po stronie API — zakładki: Przegląd, Użytkownicy, Sklepy, Oferty, Zamówienia, Kategorie, Dziennik zdarzeń
- ✅ Statystyki platformy (użytkownicy, sprzedawcy, sklepy, oferty, zamówienia wg statusu, przychód)
- ✅ Blokowanie/odblokowywanie użytkowników, sklepów i ofert (wyszukiwanie + paginacja)
- ✅ Nadawanie/odbieranie roli administratora innym kontom z poziomu panelu (bez możliwości zmiany własnych uprawnień)
- ✅ Zarządzanie kategoriami z poziomu panelu (CRUD, hierarchia, kolejność sortowania)
- ✅ Dziennik zdarzeń (`admin_audit_log`) — kto, co i kiedy zmienił w panelu

## Niezrobione jeszcze ❌

- ✅ dodac komentarze do kodu 
- ✅ Prawdziwy dostawca płatności (Stripe) — Stripe Checkout + webhook
- 4❌ Poprawic konwersje walut
- 5❌ Dodac zadania do Jira
- 3❌ zdefiniowanie systemu tlumaczenia komunikatów systemowych (po zaciagnieciu jezyka użytkownika, jezyk przegladarki, ciasteczka, po zalogowaniu predferowany jezyk i przerzucamy do ciasteczka, kazdy jezyk ma osobny plik np pl.lang en.lang czyli przerzucamy jezyk UI + komunikaty systemowe do osobnych plików)
- 2❌ przetlumaczyc komunikaty systemowe
- 2❌ Zgłaszanie ofert/sklepów przez użytkowników (kolejka moderacji) — admin musi na razie ręcznie przeglądać listy

### Funkcje z Etsy, których jeszcze nie mamy

Marketing i widoczność:
- 0❌ Promoted Listings — płatna reklama PPC wewnątrz platformy
--1❌ Offsite Ads — reklama ofert poza platformą (Google/Facebook/Instagram/Pinterest), prowizja tylko od sprzedaży
- 2❌ Kupony i wyprzedaże (zniżki %/kwotowe, zaplanowane promocje, darmowa wysyłka jako zachęta)
- 1❌ Star Seller — automatyczna odznaka zaufania z czasu odpowiedzi, terminowości wysyłki i ocen

Zakupy i płatności:
- 5❌ Personalizacja produktu — pole „dodaj swoją personalizację" na ofercie (np. grawer, dedykacja)
- 5❌ Opcje prezentowe (pakowanie, wiadomość, paragon bez ceny)
- 3❌ Filtrowanie po czasie dostawy, lokalizacji sklepu, dodatkowych atrybutach produktu

Sprzedawcy:
- 5❌ Sekcje/kolekcje w sklepie, masowa edycja ofert, import/export CSV
- 5❌ Digital downloads — natychmiastowa dostawa produktu cyfrowego bez fizycznej wysyłki
- I 5❌ Onboarding / Tutorial -

Zaufanie i obsługa sporów:
- 4❌ System zgłoszeń/sporów (case system) + program ochrony kupującego z gwarantowanym zwrotem
- R 5❌ Formalne, ustrukturyzowane polityki sklepu (zwroty, wymiany, prywatność) zamiast wolnego tekstu

Zasięg:
- 4❌ Aplikacje mobilne (obecnie tylko web) (seller/buyer)
- 5❌ PWA

LOGOWANIE:
- ❌ Logowanie przez googla

TESTOWANIE:
- ❌ zrobic k6 skrypty sprawdzic pool polaczeń

LATER:
- 0❌ Zakup i druk prawdziwych etykiet wysyłkowych z panelu (integracja z kurierem)
- 2❌ Rozbudowana analityka sprzedawcy (źródła ruchu, konwersja, wizyty) — mamy tylko podstawowe liczby
- 1❌ Automatyczne tłumaczenie treści ofert (opisy/tytuły wpisywane przez sprzedawców) i wielowalutowość — mamy wielojęzyczny interfejs (PL/EN/DE), ale treści ofert same w sobie nie są tłumaczone

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
- `POST /orders` — `{items: [{listing_id, variant_sku_id?, quantity}], shipping_addr, note}` — tworzy zamówienie(a), grupując koszyk per sklep; `variant_sku_id` jest wymagane, gdy oferta ma warianty
- `GET /orders` — historia zamówień kupującego (z pozycjami i flagą `reviewed` per pozycja)
- `GET /orders/:id` — szczegóły zamówienia
- `POST /orders/checkout-session` — `{order_ids: [...]}` — tworzy Stripe Checkout Session dla jednego lub kilku zamówień (kupujący, zamówienia muszą być `pending`); zwraca `{url}` do przekierowania
- `POST /webhooks/stripe` — webhook Stripe (bez auth, weryfikacja podpisu `Stripe-Signature`); potwierdza lub odrzuca płatność po stronie zamówień

### Sklepy
- `POST /shops` — zakłada sklep dla zalogowanego użytkownika (ustawia `is_seller = true`), wymaga auth
- `GET /shops/me` — dane własnego sklepu (auth)
- `PUT /shops/me` — edycja własnego sklepu (auth)
- `GET /shops/:id` — publiczny profil sklepu (nazwa, opis, baner, avatar, liczba ofert)

### Oferty sprzedawcy (wymagają auth + własnego sklepu)
- `POST /listings` — dodanie oferty do własnego sklepu
- `PUT /listings/:id` — edycja własnej oferty
- `DELETE /listings/:id` — trwałe usunięcie oferty; jeśli oferta ma powiązane zamówienia, zwraca `409` z podpowiedzią wyłączenia (`PUT` z `is_active: false`) zamiast usuwania
- `PUT /listings/:id/variants` — nadpisuje całą konfigurację wariantów oferty `{types: [{name, options: [...]}] (max 2), skus: [{option_values: [...], price, quantity}]}`; `has_variants` i zagregowana `quantity` oferty są przeliczane automatycznie
- `GET /seller/listings` — wszystkie własne oferty (także nieaktywne); query params: `status` (`active`/`inactive`), `category_id`, `q`, `sort` (`price_asc`/`price_desc`/`stock_asc`/`stock_desc`/`title_asc`)
- `GET /seller/listings/:id` — pełne dane własnej oferty wraz z wariantami (niezależnie od `is_active`)
- `GET /seller/shipping-profiles` / `POST /seller/shipping-profiles` / `PUT /seller/shipping-profiles/:id` / `DELETE /seller/shipping-profiles/:id` — CRUD profili wysyłki własnego sklepu (`{name, price, min_days, max_days}`)
- `GET /seller/stats` — statystyki sklepu (zamówienia, przychód, aktywne/wszystkie oferty)
- `GET /seller/orders` — zamówienia złożone we własnym sklepie
- `PUT /seller/orders/:id/status` — zmiana statusu zamówienia

### Opinie
- `POST /reviews` — `{order_item_id, rating, comment}` — ocena zakupionej pozycji (auth, wymaga statusu zamówienia `paid`/`processing`/`shipped`/`delivered`)
- `GET /listings/:id/reviews` — lista opinii dla produktu

### Wiadomości i ulubione (wymagają auth)
- `POST /messages` — `{receiver_id, body, order_id?}` — wysłanie wiadomości; odbiorca dostaje event na żywo przez WebSocket
- `GET /messages/conversations` — lista rozmów z ostatnią wiadomością i licznikiem nieprzeczytanych
- `GET /messages/unread-count` — łączny licznik nieprzeczytanych wiadomości
- `GET /messages/with/:userId` — wątek z danym użytkownikiem (oznacza wiadomości jako przeczytane)
- `GET /ws?token=` — WebSocket do powiadomień push na żywo (nowe wiadomości, zmiany licznika nieprzeczytanych); token przekazywany w query string, bo handshake WS nie pozwala na nagłówek `Authorization`
- `GET /favorites` / `GET /favorites/ids` / `POST /favorites` / `DELETE /favorites/:listingId` — ulubione produkty

### Admin (wymagają auth + `is_admin = true`)
- `GET /admin/stats` — statystyki platformy (użytkownicy, sklepy, oferty, zamówienia wg statusu, przychód)
- `GET /admin/users` — lista użytkowników; query params: `q`, `page`, `page_size`
- `PUT /admin/users/:id/status` — `{is_active}` — blokuje/odblokowuje konto
- `PUT /admin/users/:id/admin-status` — `{is_admin}` — nadaje/odbiera uprawnienia administratora (nie można zmienić własnych)
- `GET /admin/shops` — lista sklepów; query params: `q`, `page`, `page_size`
- `PUT /admin/shops/:id/status` — `{is_active}` — blokuje/odblokowuje sklep
- `GET /admin/listings` — lista ofert; query params: `q`, `active`, `page`, `page_size`
- `PUT /admin/listings/:id/status` — `{is_active}` — ukrywa/przywraca ofertę
- `GET /admin/orders` — lista wszystkich zamówień; query params: `status`, `page`, `page_size`
- `GET /admin/categories` / `POST /admin/categories` / `PUT /admin/categories/:id` / `DELETE /admin/categories/:id` — CRUD kategorii
- `GET /admin/audit-log` — historia działań administratorów; query params: `page`, `page_size`

**Nadanie pierwszego konta administratora** (kolejnym kontom uprawnienia nadaje się już z panelu — zakładka Użytkownicy):
```sql
UPDATE users SET is_admin = TRUE WHERE email = 'twoj@email.pl';
```

## Environment Variables

```
PGUSER=admin
PGPASSWORD=***
PGHOST=...neon.tech
PGPORT=5432
PGDATABASE=flipthetable
JWT_SECRET=***           # zmienić na prod!
PORT=8080

# Stripe (płatności)
STRIPE_SECRET_KEY=***
STRIPE_PUBLISHABLE_KEY=***
STRIPE_WEBHOOK_SECRET=***   # z `stripe listen` lokalnie, albo z ustawień webhooka w Dashboardzie na prod
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
