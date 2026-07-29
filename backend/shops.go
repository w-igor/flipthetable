package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Shop struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	BannerURL   string    `json:"banner_url,omitempty"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	IsActive    bool      `json:"is_active"`
	SalesCount  int       `json:"sales_count"`
	CreatedAt   time.Time `json:"created_at"`
}

func handleGetShop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var shop Shop
	err := db.QueryRow(
		"SELECT id, owner_id, name, slug, description, banner_url, avatar_url, is_active, sales_count, created_at FROM shops WHERE id = $1",
		id,
	).Scan(&shop.ID, &shop.OwnerID, &shop.Name, &shop.Slug, &shop.Description, &shop.BannerURL, &shop.AvatarURL, &shop.IsActive, &shop.SalesCount, &shop.CreatedAt)

	if err != nil {
		http.Error(w, "Shop not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shop)
}

func handleCreateShop(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	shopID := uuid.New().String()

	_, err := db.Exec(
		"INSERT INTO shops (id, owner_id, name, slug, description) VALUES ($1, $2, $3, $4, $5)",
		shopID, userID, req.Name, req.Slug, req.Description,
	)

	if err != nil {
		http.Error(w, "Failed to create shop", http.StatusInternalServerError)
		return
	}

	// Mark user as seller
	db.Exec("UPDATE users SET is_seller = true WHERE id = $1", userID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": shopID, "message": "Shop created"})
}
