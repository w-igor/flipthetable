package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Order struct {
	ID        int         `json:"id"`
	UserID    int         `json:"user_id"`
	TotalPrice float64    `json:"total_price"`
	Status    string      `json:"status"`
	Items     []OrderItem `json:"items,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID              int     `json:"id"`
	OrderID         int     `json:"order_id"`
	ProductID       int     `json:"product_id"`
	Quantity        int     `json:"quantity"`
	PriceAtPurchase float64 `json:"price_at_purchase"`
	Product         Product `json:"product,omitempty"`
}

type CreateOrderRequest struct {
	Items []struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
	} `json:"items"`
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get user ID
	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Calculate total and validate products
	var totalPrice float64
	orderItems := make([]OrderItem, 0)

	for _, item := range req.Items {
		var priceAtPurchase float64
		var stock int

		err := tx.QueryRow(
			"SELECT price, stock FROM products WHERE id = $1",
			item.ProductID,
		).Scan(&priceAtPurchase, &stock)

		if err != nil {
			http.Error(w, "Product not found", http.StatusBadRequest)
			return
		}

		if stock < item.Quantity {
			http.Error(w, "Insufficient stock", http.StatusBadRequest)
			return
		}

		totalPrice += priceAtPurchase * float64(item.Quantity)

		orderItems = append(orderItems, OrderItem{
			ProductID:       item.ProductID,
			Quantity:        item.Quantity,
			PriceAtPurchase: priceAtPurchase,
		})
	}

	// Create order
	var orderID int
	err = tx.QueryRow(
		"INSERT INTO orders (user_id, total_price, status) VALUES ($1, $2, 'pending') RETURNING id",
		userID, totalPrice,
	).Scan(&orderID)

	if err != nil {
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	// Add order items and update stock
	for _, item := range orderItems {
		_, err := tx.Exec(
			"INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase) VALUES ($1, $2, $3, $4)",
			orderID, item.ProductID, item.Quantity, item.PriceAtPurchase,
		)

		if err != nil {
			http.Error(w, "Failed to create order items", http.StatusInternalServerError)
			return
		}

		// Reduce stock
		_, err = tx.Exec(
			"UPDATE products SET stock = stock - $1 WHERE id = $2",
			item.Quantity, item.ProductID,
		)

		if err != nil {
			http.Error(w, "Failed to update stock", http.StatusInternalServerError)
			return
		}
	}

	// Clear user's cart
	_, err = tx.Exec("DELETE FROM cart_items WHERE user_id = $1", userID)
	if err != nil {
		http.Error(w, "Failed to clear cart", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           orderID,
		"total_price":  totalPrice,
		"status":       "pending",
		"created_at":   time.Now(),
	})
}

func handleGetOrders(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	rows, err := db.Query(`
		SELECT id, user_id, total_price, status, created_at, updated_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
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

		// Get order items
		itemRows, err := db.Query(`
			SELECT oi.id, oi.order_id, oi.product_id, oi.quantity, oi.price_at_purchase,
				   p.id, p.category_id, p.name, p.description, p.price, p.stock, p.image_url, p.seller_id, p.rating, p.reviews_count
			FROM order_items oi
			JOIN products p ON oi.product_id = p.id
			WHERE oi.order_id = $1
		`, order.ID)

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

func handleGetOrder(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")
	orderID := r.PathValue("id")

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var order Order
	err = db.QueryRow(`
		SELECT id, user_id, total_price, status, created_at, updated_at
		FROM orders
		WHERE id = $1 AND user_id = $2
	`, orderID, userID).Scan(&order.ID, &order.UserID, &order.TotalPrice, &order.Status, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Get order items
	rows, err := db.Query(`
		SELECT oi.id, oi.order_id, oi.product_id, oi.quantity, oi.price_at_purchase,
			   p.id, p.category_id, p.name, p.description, p.price, p.stock, p.image_url, p.seller_id, p.rating, p.reviews_count
		FROM order_items oi
		JOIN products p ON oi.product_id = p.id
		WHERE oi.order_id = $1
	`, order.ID)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item OrderItem
			var product Product
			if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.PriceAtPurchase,
				&product.ID, &product.CategoryID, &product.Name, &product.Description, &product.Price, &product.Stock, &product.ImageURL, &product.SellerID, &product.Rating, &product.ReviewsCount); err == nil {
				item.Product = product
				order.Items = append(order.Items, item)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func handleUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	validStatuses := map[string]bool{"pending": true, "confirmed": true, "shipped": true, "delivered": true, "cancelled": true}
	if !validStatuses[req.Status] {
		http.Error(w, "Invalid status", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(
		"UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2",
		req.Status, orderID,
	)

	if err != nil {
		http.Error(w, "Failed to update order", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Order updated", "status": req.Status})
}

func handleGetOrderStats(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var stats struct {
		TotalOrders    int     `json:"total_orders"`
		TotalSpent     float64 `json:"total_spent"`
		AvgOrderValue  float64 `json:"avg_order_value"`
		PendingOrders  int     `json:"pending_orders"`
		DeliveredCount int     `json:"delivered_count"`
	}

	// Get stats
	err = db.QueryRow(`
		SELECT
			COUNT(*),
			COALESCE(SUM(total_price), 0),
			COALESCE(AVG(total_price), 0),
			COUNT(CASE WHEN status = 'pending' THEN 1 END),
			COUNT(CASE WHEN status = 'delivered' THEN 1 END)
		FROM orders
		WHERE user_id = $1
	`, userID).Scan(&stats.TotalOrders, &stats.TotalSpent, &stats.AvgOrderValue, &stats.PendingOrders, &stats.DeliveredCount)

	if err != nil && err != sql.ErrNoRows {
		log.Println("Error getting order stats:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
