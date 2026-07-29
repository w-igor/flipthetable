package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Slug        string `json:"slug"`
}

type Product struct {
	ID           int     `json:"id"`
	CategoryID   int     `json:"category_id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	Stock        int     `json:"stock"`
	ImageURL     string  `json:"image_url"`
	SellerID     int     `json:"seller_id"`
	Rating       float64 `json:"rating"`
	ReviewsCount int     `json:"reviews_count"`
}

type CartItem struct {
	ID        int     `json:"id"`
	ProductID int     `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Product   Product `json:"product,omitempty"`
}

func handleGetCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, description, slug FROM categories")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Description, &cat.Slug); err != nil {
			continue
		}
		categories = append(categories, cat)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

func handleGetProducts(w http.ResponseWriter, r *http.Request) {
	categoryID := r.URL.Query().Get("category_id")
	search := r.URL.Query().Get("search")
	limit := r.URL.Query().Get("limit")
	offset := r.URL.Query().Get("offset")

	if limit == "" {
		limit = "20"
	}
	if offset == "" {
		offset = "0"
	}

	query := "SELECT id, category_id, name, description, price, stock, image_url, seller_id, rating, reviews_count FROM products WHERE stock > 0"
	var args []interface{}

	if categoryID != "" {
		query += " AND category_id = $" + strconv.Itoa(len(args)+1)
		args = append(args, categoryID)
	}

	if search != "" {
		query += " AND (name ILIKE $" + strconv.Itoa(len(args)+1) + " OR description ILIKE $" + strconv.Itoa(len(args)+2) + ")"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURL, &p.SellerID, &p.Rating, &p.ReviewsCount); err != nil {
			continue
		}
		products = append(products, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
}

func handleGetProduct(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var p Product
	err := db.QueryRow(
		"SELECT id, category_id, name, description, price, stock, image_url, seller_id, rating, reviews_count FROM products WHERE id = $1",
		id,
	).Scan(&p.ID, &p.CategoryID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.ImageURL, &p.SellerID, &p.Rating, &p.ReviewsCount)

	if err != nil {
		http.Error(w, "Product not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

func handleGetCart(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	rows, err := db.Query(`
		SELECT ci.id, ci.product_id, ci.quantity,
		       p.id, p.category_id, p.name, p.description, p.price, p.stock, p.image_url, p.seller_id, p.rating, p.reviews_count
		FROM cart_items ci
		JOIN products p ON ci.product_id = p.id
		WHERE ci.user_id = $1
	`, userID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []CartItem
	for rows.Next() {
		var item CartItem
		var product Product
		if err := rows.Scan(&item.ID, &item.ProductID, &item.Quantity,
			&product.ID, &product.CategoryID, &product.Name, &product.Description, &product.Price, &product.Stock, &product.ImageURL, &product.SellerID, &product.Rating, &product.ReviewsCount); err != nil {
			continue
		}
		item.Product = product
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func handleAddToCart(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var req struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
	}
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

	if req.Quantity <= 0 {
		http.Error(w, "Quantity must be positive", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`
		INSERT INTO cart_items (user_id, product_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, product_id)
		DO UPDATE SET quantity = cart_items.quantity + $3
	`, userID, req.ProductID, req.Quantity)

	if err != nil {
		http.Error(w, "Failed to add to cart", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Added to cart"})
}

func handleRemoveFromCart(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")
	cartID := r.PathValue("id")

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	_, err = db.Exec(
		"DELETE FROM cart_items WHERE id = $1 AND user_id = $2",
		cartID, userID,
	)

	if err != nil {
		http.Error(w, "Failed to remove from cart", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Removed from cart"})
}

func handleUpdateCartItem(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")
	cartID := r.PathValue("id")

	var req struct {
		Quantity int `json:"quantity"`
	}
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

	if req.Quantity <= 0 {
		http.Error(w, "Quantity must be positive", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(
		"UPDATE cart_items SET quantity = $1 WHERE id = $2 AND user_id = $3",
		req.Quantity, cartID, userID,
	)

	if err != nil {
		http.Error(w, "Failed to update cart", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Cart updated"})
}
