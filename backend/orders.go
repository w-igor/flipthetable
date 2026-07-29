package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID            string      `json:"id"`
	BuyerID       string      `json:"buyer_id"`
	ShopID        string      `json:"shop_id"`
	Status        string      `json:"status"`
	TotalAmount   float64     `json:"total_amount"`
	Currency      string      `json:"currency"`
	ShippingAddr  Address     `json:"shipping_addr"`
	Note          string      `json:"note,omitempty"`
	Items         []OrderItem `json:"items,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID              string  `json:"id"`
	OrderID         string  `json:"order_id"`
	ListingID       string  `json:"listing_id"`
	Quantity        int     `json:"quantity"`
	UnitPrice       float64 `json:"unit_price"`
	TitleSnapshot   string  `json:"title_snapshot"`
	ListingDetail   Listing `json:"listing,omitempty"`
}

type Address struct {
	FullName   string `json:"full_name"`
	Address    string `json:"address"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	Phone      string `json:"phone"`
}

// JSONB support for PostgreSQL
func (a Address) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *Address) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &a)
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	var req struct {
		ShopID       string  `json:"shop_id"`
		Items        []struct {
			ListingID string `json:"listing_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
		ShippingAddr Address `json:"shipping_addr"`
		Note         string  `json:"note"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var totalAmount float64
	orderItems := make([]OrderItem, 0)

	for _, item := range req.Items {
		var price float64
		var quantity int

		err := tx.QueryRow(
			"SELECT price, quantity FROM listings WHERE id = $1",
			item.ListingID,
		).Scan(&price, &quantity)

		if err != nil {
			http.Error(w, "Listing not found", http.StatusBadRequest)
			return
		}

		if quantity < item.Quantity {
			http.Error(w, "Insufficient stock", http.StatusBadRequest)
			return
		}

		totalAmount += price * float64(item.Quantity)
		orderItems = append(orderItems, OrderItem{
			ListingID:     item.ListingID,
			Quantity:      item.Quantity,
			UnitPrice:     price,
		})
	}

	orderID := uuid.New().String()
	shippingJSON, _ := json.Marshal(req.ShippingAddr)

	err = tx.QueryRow(
		"INSERT INTO orders (id, buyer_id, shop_id, total_amount, shipping_addr, note) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
		orderID, userID, req.ShopID, totalAmount, shippingJSON, req.Note,
	).Scan(&orderID)

	if err != nil {
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	for _, item := range orderItems {
		itemID := uuid.New().String()
		_, err := tx.Exec(
			"INSERT INTO order_items (id, order_id, listing_id, quantity, unit_price, title_snapshot) VALUES ($1, $2, $3, $4, $5, (SELECT title FROM listings WHERE id = $6))",
			itemID, orderID, item.ListingID, item.Quantity, item.UnitPrice, item.ListingID,
		)

		if err != nil {
			http.Error(w, "Failed to create order items", http.StatusInternalServerError)
			return
		}

		tx.Exec("UPDATE listings SET quantity = quantity - $1 WHERE id = $2", item.Quantity, item.ListingID)
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":            orderID,
		"total_amount":  totalAmount,
		"status":        "pending",
		"created_at":    time.Now(),
	})
}

func handleGetOrders(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	rows, err := db.Query(`
		SELECT id, buyer_id, shop_id, status, total_amount, currency, shipping_addr, note, created_at, updated_at
		FROM orders
		WHERE buyer_id = $1
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
		var shippingJSON []byte

		if err := rows.Scan(&order.ID, &order.BuyerID, &order.ShopID, &order.Status, &order.TotalAmount, &order.Currency, &shippingJSON, &order.Note, &order.CreatedAt, &order.UpdatedAt); err != nil {
			continue
		}

		json.Unmarshal(shippingJSON, &order.ShippingAddr)
		orders = append(orders, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func handleGetOrder(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	orderID := r.PathValue("id")

	var order Order
	var shippingJSON []byte

	err := db.QueryRow(`
		SELECT id, buyer_id, shop_id, status, total_amount, currency, shipping_addr, note, created_at, updated_at
		FROM orders
		WHERE id = $1 AND buyer_id = $2
	`, orderID, userID).Scan(&order.ID, &order.BuyerID, &order.ShopID, &order.Status, &order.TotalAmount, &order.Currency, &shippingJSON, &order.Note, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	json.Unmarshal(shippingJSON, &order.ShippingAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
