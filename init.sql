-- ============================================================
--  ETSY-LIKE MARKETPLACE — PostgreSQL DDL
--  Wersja: 1.0 | Baza: PostgreSQL 15+
-- ============================================================

-- ── TYPY ENUMEROWANE ────────────────────────────────────────

CREATE TYPE order_status AS ENUM (
  'pending',
  'paid',
  'processing',
  'shipped',
  'delivered',
  'cancelled',
  'refunded'
);

CREATE TYPE payment_status AS ENUM (
  'pending',
  'completed',
  'failed',
  'refunded'
);

-- ============================================================
--  1. USERS
--     Kupujący i sprzedający — jedna tabela, flaga is_seller
-- ============================================================
CREATE TABLE users (
  id            UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
  email         VARCHAR(255)      NOT NULL UNIQUE,
  username      VARCHAR(100)      NOT NULL UNIQUE,
  full_name     VARCHAR(200),
  avatar_url    TEXT,
  password_hash TEXT              NOT NULL,
  is_seller     BOOLEAN           NOT NULL DEFAULT FALSE,
  is_active     BOOLEAN           NOT NULL DEFAULT TRUE,
  created_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email    ON users (email);
CREATE INDEX idx_users_username ON users (username);

-- ============================================================
--  2. SHOPS
--     Każdy sprzedający może mieć jeden sklep
-- ============================================================
CREATE TABLE shops (
  id            UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id      UUID              NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name          VARCHAR(100)      NOT NULL,
  slug          VARCHAR(100)      NOT NULL UNIQUE,
  description   TEXT,
  banner_url    TEXT,
  avatar_url    TEXT,
  is_active     BOOLEAN           NOT NULL DEFAULT TRUE,
  sales_count   INTEGER           NOT NULL DEFAULT 0,
  created_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shops_owner_id ON shops (owner_id);
CREATE INDEX idx_shops_slug     ON shops (slug);

-- ============================================================
--  3. CATEGORIES
--     Hierarchia kategorii (self-join przez parent_id)
-- ============================================================
CREATE TABLE categories (
  id            UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_id     UUID              REFERENCES categories(id) ON DELETE SET NULL,
  name          VARCHAR(100)      NOT NULL,
  slug          VARCHAR(100)      NOT NULL UNIQUE,
  description   TEXT,
  sort_order    INTEGER           NOT NULL DEFAULT 0
);

CREATE INDEX idx_categories_parent_id ON categories (parent_id);
CREATE INDEX idx_categories_slug      ON categories (slug);

-- ============================================================
--  4. LISTINGS
--     Produkty wystawione w sklepach
-- ============================================================
CREATE TABLE listings (
  id            UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
  shop_id       UUID              NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
  category_id   UUID              REFERENCES categories(id) ON DELETE SET NULL,
  title         VARCHAR(255)      NOT NULL,
  description   TEXT,
  price         NUMERIC(10, 2)    NOT NULL CHECK (price >= 0),
  currency      CHAR(3)           NOT NULL DEFAULT 'PLN',
  quantity      INTEGER           NOT NULL DEFAULT 1 CHECK (quantity >= 0),
  is_active     BOOLEAN           NOT NULL DEFAULT TRUE,
  views_count   INTEGER           NOT NULL DEFAULT 0,
  sales_count   INTEGER           NOT NULL DEFAULT 0,
  avg_rating    NUMERIC(3, 2),
  created_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
  updated_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_listings_shop_id     ON listings (shop_id);
CREATE INDEX idx_listings_category_id ON listings (category_id);
CREATE INDEX idx_listings_is_active   ON listings (is_active);
CREATE INDEX idx_listings_price       ON listings (price);

-- ============================================================
--  5. LISTING_PHOTOS
--     Zdjęcia produktu (jedno główne + galeria)
-- ============================================================
CREATE TABLE listing_photos (
  id            UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
  listing_id    UUID              NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  url           TEXT              NOT NULL,
  alt_text      VARCHAR(255),
  is_primary    BOOLEAN           NOT NULL DEFAULT FALSE,
  sort_order    INTEGER           NOT NULL DEFAULT 0
);

CREATE INDEX idx_listing_photos_listing_id ON listing_photos (listing_id);

-- ============================================================
--  6. TAGS
--     Słowa kluczowe do filtrowania i wyszukiwania
-- ============================================================
CREATE TABLE tags (
  id            UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
  name          VARCHAR(50)       NOT NULL UNIQUE
);

CREATE INDEX idx_tags_name ON tags (name);

-- ============================================================
--  7. LISTING_TAGS
--     Relacja wiele-do-wielu: listings ↔ tags
-- ============================================================
CREATE TABLE listing_tags (
  listing_id    UUID              NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  tag_id        UUID              NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  PRIMARY KEY (listing_id, tag_id)
);

CREATE INDEX idx_listing_tags_tag_id ON listing_tags (tag_id);

-- ============================================================
--  8. ORDERS
--     Zamówienie kupującego (może obejmować jeden sklep)
-- ============================================================
CREATE TABLE orders (
  id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
  buyer_id        UUID            NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  shop_id         UUID            NOT NULL REFERENCES shops(id) ON DELETE RESTRICT,
  status          order_status    NOT NULL DEFAULT 'pending',
  total_amount    NUMERIC(10, 2)  NOT NULL CHECK (total_amount >= 0),
  currency        CHAR(3)         NOT NULL DEFAULT 'PLN',
  shipping_addr   JSONB           NOT NULL,
  note            TEXT,
  created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_buyer_id  ON orders (buyer_id);
CREATE INDEX idx_orders_shop_id   ON orders (shop_id);
CREATE INDEX idx_orders_status    ON orders (status);
CREATE INDEX idx_orders_created_at ON orders (created_at DESC);

-- ============================================================
--  9. ORDER_ITEMS
--     Pozycje w zamówieniu (snapshot ceny w chwili zakupu)
-- ============================================================
CREATE TABLE order_items (
  id            UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id      UUID              NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
  listing_id    UUID              NOT NULL REFERENCES listings(id) ON DELETE RESTRICT,
  quantity      INTEGER           NOT NULL CHECK (quantity > 0),
  unit_price    NUMERIC(10, 2)    NOT NULL CHECK (unit_price >= 0),
  title_snapshot VARCHAR(255)     NOT NULL  -- kopia tytułu na chwilę zakupu
);

CREATE INDEX idx_order_items_order_id   ON order_items (order_id);
CREATE INDEX idx_order_items_listing_id ON order_items (listing_id);

-- ============================================================
--  10. PAYMENTS
--      Płatności powiązane z zamówieniami
-- ============================================================
CREATE TABLE payments (
  id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id        UUID            NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
  provider        VARCHAR(50)     NOT NULL,   -- 'stripe', 'przelewy24', itp.
  provider_tx_id  VARCHAR(255),               -- ID transakcji po stronie providera
  status          payment_status  NOT NULL DEFAULT 'pending',
  amount          NUMERIC(10, 2)  NOT NULL CHECK (amount >= 0),
  currency        CHAR(3)         NOT NULL DEFAULT 'PLN',
  paid_at         TIMESTAMPTZ,
  created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_payments_order_id ON payments (order_id);
CREATE INDEX idx_payments_status   ON payments (status);

-- ============================================================
--  11. REVIEWS
--      Opinie wystawiane po zakupie
-- ============================================================
CREATE TABLE reviews (
  id              UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
  listing_id      UUID            NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  reviewer_id     UUID            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  order_item_id   UUID            NOT NULL REFERENCES order_items(id) ON DELETE RESTRICT,
  rating          SMALLINT        NOT NULL CHECK (rating BETWEEN 1 AND 5),
  comment         TEXT,
  created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
  UNIQUE (reviewer_id, order_item_id)  -- jedna opinia na pozycję zamówienia
);

CREATE INDEX idx_reviews_listing_id  ON reviews (listing_id);
CREATE INDEX idx_reviews_reviewer_id ON reviews (reviewer_id);
CREATE INDEX idx_reviews_rating      ON reviews (rating);

-- ============================================================
--  12. FAVORITES
--      Produkty zapisane przez użytkownika (serce/polub)
-- ============================================================
CREATE TABLE favorites (
  user_id       UUID              NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  listing_id    UUID              NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  created_at    TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, listing_id)
);

CREATE INDEX idx_favorites_listing_id ON favorites (listing_id);

-- ============================================================
--  13. MESSAGES
--      Wiadomości między kupującymi a sprzedającymi
-- ============================================================
CREATE TABLE messages (
  id            UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
  sender_id     UUID              NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  receiver_id   UUID              NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  order_id      UUID              REFERENCES orders(id) ON DELETE SET NULL,
  body          TEXT              NOT NULL,
  is_read       BOOLEAN           NOT NULL DEFAULT FALSE,
  sent_at       TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
  CHECK (sender_id <> receiver_id)
);

CREATE INDEX idx_messages_sender_id   ON messages (sender_id);
CREATE INDEX idx_messages_receiver_id ON messages (receiver_id);
CREATE INDEX idx_messages_sent_at     ON messages (sent_at DESC);

-- ============================================================
--  TRIGGER: auto-aktualizacja updated_at
-- ============================================================
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON users
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_shops_updated_at
  BEFORE UPDATE ON shops
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_listings_updated_at
  BEFORE UPDATE ON listings
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_orders_updated_at
  BEFORE UPDATE ON orders
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================
--  KOMENTARZE (opcjonalne, ale pomocne w pg_dump / dokumentacji)
-- ============================================================
COMMENT ON TABLE users        IS 'Kupujący i sprzedający na platformie';
COMMENT ON TABLE shops        IS 'Sklepy prowadzone przez sprzedających';
COMMENT ON TABLE categories   IS 'Hierarchia kategorii produktów (self-join)';
COMMENT ON TABLE listings     IS 'Produkty wystawione w sklepach';
COMMENT ON TABLE listing_photos IS 'Zdjęcia produktu (jedno główne + galeria)';
COMMENT ON TABLE tags         IS 'Słowa kluczowe do wyszukiwania';
COMMENT ON TABLE listing_tags IS 'Relacja M:N — produkty i tagi';
COMMENT ON TABLE orders       IS 'Zamówienia złożone przez kupujących';
COMMENT ON TABLE order_items  IS 'Pozycje w zamówieniu ze snapshotem ceny';
COMMENT ON TABLE payments     IS 'Płatności za zamówienia';
COMMENT ON TABLE reviews      IS 'Opinie wystawiane po dokonaniu zakupu';
COMMENT ON TABLE favorites    IS 'Zapisane/polubione produkty użytkownika';
COMMENT ON TABLE messages     IS 'Wiadomości między użytkownikami';