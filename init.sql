-- For Neon (cloud PostgreSQL), use existing database
-- Run these queries in your Neon console or via psql

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stage 2: Shop & Catalog
CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    slug VARCHAR(100) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2) NOT NULL,
    stock INTEGER DEFAULT 0,
    image_url VARCHAR(500),
    seller_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating DECIMAL(3, 2) DEFAULT 0,
    reviews_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS cart_items (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 1,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, product_id)
);

CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_price DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id INTEGER NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL,
    price_at_purchase DECIMAL(10, 2) NOT NULL
);

-- Indexes
-- Stage 4: WebSockets & Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_email VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT,
    order_id INTEGER,
    read BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_seller_id ON products(seller_id);
CREATE INDEX idx_cart_items_user_id ON cart_items(user_id);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_notifications_user_email ON notifications(user_email);
CREATE INDEX idx_notifications_created_at ON notifications(created_at);

-- Sample admin user (password: admin123)
INSERT INTO users (email, password) VALUES
    ('admin@example.com', '$2a$10$xvlVsKq7W8v8YG0J/Dy0sOlA.zf0/V0qYu0qB.K3SN7K.bFzKTJF6')
ON CONFLICT DO NOTHING;

-- Sample data
INSERT INTO categories (name, description, slug) VALUES
    ('Electronics', 'Electronic devices and gadgets', 'electronics'),
    ('Clothing', 'Apparel and fashion items', 'clothing'),
    ('Home', 'Home and garden products', 'home'),
    ('Art', 'Handmade art and crafts', 'art')
ON CONFLICT DO NOTHING;

INSERT INTO products (category_id, name, description, price, stock, seller_id, image_url) VALUES
    (1, 'Vintage Camera', 'Beautiful retro 35mm camera', 89.99, 5, 1, 'https://via.placeholder.com/200?text=Camera'),
    (1, 'Wireless Headphones', 'Bluetooth noise-canceling', 129.99, 10, 1, 'https://via.placeholder.com/200?text=Headphones'),
    (2, 'Handmade Scarf', 'Wool knit scarf in natural colors', 34.99, 20, 1, 'https://via.placeholder.com/200?text=Scarf'),
    (3, 'Ceramic Vase', 'Hand-thrown ceramic vase', 45.99, 8, 1, 'https://via.placeholder.com/200?text=Vase'),
    (4, 'Abstract Painting', 'Modern abstract oil painting', 199.99, 1, 1, 'https://via.placeholder.com/200?text=Painting')
ON CONFLICT DO NOTHING;
