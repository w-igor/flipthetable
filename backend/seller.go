package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

type SellerProfile struct {
	UserID           int       `json:"user_id"`
	Email            string    `json:"email"`
	SellerName       string    `json:"seller_name"`
	SellerDescription string   `json:"seller_description"`
	SellerVerified   bool      `json:"seller_verified"`
	ProductCount     int       `json:"product_count"`
	TotalSales       float64   `json:"total_sales"`
	AverageRating    float64   `json:"average_rating"`
	JoinedAt         time.Time `json:"joined_at"`
}

type ProductInput struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	CategoryID  int     `json:"category_id"`
	ImageURL    string  `json:"image_url"`
}

// Seller registration
func handleRegisterAsSeller(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var req struct {
		SellerName        string `json:"seller_name"`
		SellerDescription string `json:"seller_description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.SellerName == "" {
		http.Error(w, "Seller name required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(
		"UPDATE users SET is_seller = true, seller_name = $1, seller_description = $2 WHERE email = $3",
		req.SellerName, req.SellerDescription, email,
	)

	if err != nil {
		http.Error(w, "Failed to register as seller", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Registered as seller"})
}

// Get seller profile
func handleGetSellerProfile(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var profile SellerProfile
	var userID int

	err := db.QueryRow(
		"SELECT id, email, seller_name, seller_description, seller_verified, created_at FROM users WHERE email = $1",
		email,
	).Scan(&userID, &profile.Email, &profile.SellerName, &profile.SellerDescription, &profile.SellerVerified, &profile.JoinedAt)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	profile.UserID = userID

	// Get product count
	db.QueryRow("SELECT COUNT(*) FROM products WHERE seller_id = $1", userID).Scan(&profile.ProductCount)

	// Get total sales
	db.QueryRow(`
		SELECT COALESCE(SUM(total_price), 0) FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		JOIN products p ON oi.product_id = p.id
		WHERE p.seller_id = $1
	`, userID).Scan(&profile.TotalSales)

	// Get average rating
	db.QueryRow(
		"SELECT COALESCE(AVG(rating), 0) FROM products WHERE seller_id = $1",
		userID,
	).Scan(&profile.AverageRating)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// Get seller's products
func handleGetSellerProducts(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	rows, err := db.Query(`
		SELECT id, category_id, name, description, price, stock, image_url, seller_id, rating, reviews_count, created_at
		FROM products
		WHERE seller_id = $1
		ORDER BY created_at DESC
	`, userID)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ProductWithDate struct {
		Product
		CreatedAt time.Time `json:"created_at"`
	}

	var products []ProductWithDate
	for rows.Next() {
		var p ProductWithDate
		if err := rows.Scan(&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURL, &p.SellerID, &p.Rating, &p.ReviewsCount, &p.CreatedAt); err != nil {
			continue
		}
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

// Create product
func handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var req ProductInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Price <= 0 || req.CategoryID == 0 {
		http.Error(w, "Invalid product data", http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1 AND is_seller = true", email).Scan(&userID)
	if err != nil {
		http.Error(w, "Not a seller", http.StatusForbidden)
		return
	}

	var productID int
	err = db.QueryRow(`
		INSERT INTO products (category_id, name, description, price, stock, image_url, seller_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, req.CategoryID, req.Name, req.Description, req.Price, req.Stock, req.ImageURL, userID).Scan(&productID)

	if err != nil {
		http.Error(w, "Failed to create product", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      productID,
		"message": "Product created",
	})
}

// Update product
func handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")
	productID := r.PathValue("id")

	var req ProductInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	var ownerID int
	err = db.QueryRow("SELECT seller_id FROM products WHERE id = $1", productID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	_, err = db.Exec(`
		UPDATE products
		SET name = $1, description = $2, price = $3, stock = $4, category_id = $5, image_url = $6, updated_at = NOW()
		WHERE id = $7
	`, req.Name, req.Description, req.Price, req.Stock, req.CategoryID, req.ImageURL, productID)

	if err != nil {
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Product updated"})
}

// Delete product
func handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")
	productID := r.PathValue("id")

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	var ownerID int
	err = db.QueryRow("SELECT seller_id FROM products WHERE id = $1", productID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	_, err = db.Exec("DELETE FROM products WHERE id = $1", productID)
	if err != nil {
		http.Error(w, "Failed to delete product", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Product deleted"})
}

// Get seller's orders
func handleGetSellerOrders(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	rows, err := db.Query(`
		SELECT DISTINCT o.id, o.user_id, o.total_price, o.status, o.created_at, o.updated_at
		FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		JOIN products p ON oi.product_id = p.id
		WHERE p.seller_id = $1
		ORDER BY o.created_at DESC
	`, userID)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.UserID, &order.TotalPrice, &order.Status, &order.CreatedAt, &order.UpdatedAt); err != nil {
			continue
		}

		// Get order items for this seller
		itemRows, err := db.Query(`
			SELECT oi.id, oi.order_id, oi.product_id, oi.quantity, oi.price_at_purchase,
				   p.id, p.category_id, p.name, p.description, p.price, p.stock, p.image_url, p.seller_id, p.rating, p.reviews_count
			FROM order_items oi
			JOIN products p ON oi.product_id = p.id
			WHERE oi.order_id = $1 AND p.seller_id = $2
		`, order.ID, userID)

		if err == nil {
			defer itemRows.Close()
			for itemRows.Next() {
				var item OrderItem
				var product Product
				if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.PriceAtPurchase,
					&product.ID, &product.CategoryID, &product.Name, &product.Description, &product.Price, &product.Stock, &product.ImageURL, &product.SellerID, &product.Rating, &product.ReviewsCount); err == nil {
					item.Product = product
					order.Items = append(order.Items, item)
				}
			}
		}

		orders = append(orders, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// Get seller analytics/stats
func handleGetSellerStats(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var stats struct {
		TotalProducts    int     `json:"total_products"`
		TotalSales       float64 `json:"total_sales"`
		TotalOrders      int     `json:"total_orders"`
		AverageRating    float64 `json:"average_rating"`
		InventoryValue   float64 `json:"inventory_value"`
		OrdersThisMonth  int     `json:"orders_this_month"`
		SalesThisMonth   float64 `json:"sales_this_month"`
		TopProduct       string  `json:"top_product"`
		PendingOrders    int     `json:"pending_orders"`
	}

	// Total products
	db.QueryRow("SELECT COUNT(*) FROM products WHERE seller_id = $1", userID).Scan(&stats.TotalProducts)

	// Total sales
	db.QueryRow(`
		SELECT COALESCE(SUM(total_price), 0) FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		JOIN products p ON oi.product_id = p.id
		WHERE p.seller_id = $1
	`, userID).Scan(&stats.TotalSales)

	// Total orders (distinct)
	db.QueryRow(`
		SELECT COUNT(DISTINCT o.id) FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		JOIN products p ON oi.product_id = p.id
		WHERE p.seller_id = $1
	`, userID).Scan(&stats.TotalOrders)

	// Average rating
	db.QueryRow(
		"SELECT COALESCE(AVG(rating), 0) FROM products WHERE seller_id = $1",
		userID,
	).Scan(&stats.AverageRating)

	// Inventory value
	db.QueryRow(
		"SELECT COALESCE(SUM(price * stock), 0) FROM products WHERE seller_id = $1",
		userID,
	).Scan(&stats.InventoryValue)

	// This month sales
	db.QueryRow(`
		SELECT COUNT(DISTINCT o.id), COALESCE(SUM(total_price), 0)
		FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		JOIN products p ON oi.product_id = p.id
		WHERE p.seller_id = $1 AND o.created_at >= NOW() - INTERVAL '30 days'
	`, userID).Scan(&stats.OrdersThisMonth, &stats.SalesThisMonth)

	// Top product
	db.QueryRow(`
		SELECT p.name
		FROM products p
		WHERE p.seller_id = $1
		ORDER BY p.reviews_count DESC
		LIMIT 1
	`, userID).Scan(&stats.TopProduct)

	// Pending orders
	db.QueryRow(`
		SELECT COUNT(DISTINCT o.id) FROM orders o
		JOIN order_items oi ON o.id = oi.order_id
		JOIN products p ON oi.product_id = p.id
		WHERE p.seller_id = $1 AND o.status = 'pending'
	`, userID).Scan(&stats.PendingOrders)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Update product stock
func handleUpdateProductStock(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")
	productID := r.PathValue("id")

	var req struct {
		Stock int `json:"stock"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Stock < 0 {
		http.Error(w, "Stock cannot be negative", http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Verify ownership
	var ownerID int
	err = db.QueryRow("SELECT seller_id FROM products WHERE id = $1", productID).Scan(&ownerID)
	if err != nil || ownerID != userID {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	_, err = db.Exec("UPDATE products SET stock = $1, updated_at = NOW() WHERE id = $2", req.Stock, productID)
	if err != nil {
		http.Error(w, "Failed to update stock", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Stock updated"})
}
