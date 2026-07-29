package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Listing struct {
	ID           string    `json:"id"`
	ShopID       string    `json:"shop_id"`
	CategoryID   string    `json:"category_id,omitempty"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Price        float64   `json:"price"`
	Currency     string    `json:"currency"`
	Quantity     int       `json:"quantity"`
	IsActive     bool      `json:"is_active"`
	ViewsCount   int       `json:"views_count"`
	SalesCount   int       `json:"sales_count"`
	AvgRating    float64   `json:"avg_rating,omitempty"`
	Photos       []Photo   `json:"photos,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Photo struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	AltText   string `json:"alt_text,omitempty"`
	IsPrimary bool   `json:"is_primary"`
}

func handleGetListings(w http.ResponseWriter, r *http.Request) {
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

	query := "SELECT id, shop_id, category_id, title, description, price, currency, quantity, is_active, views_count, sales_count, avg_rating, created_at, updated_at FROM listings WHERE is_active = true"
	var args []interface{}

	if categoryID != "" {
		query += " AND category_id = $" + strconv.Itoa(len(args)+1)
		args = append(args, categoryID)
	}

	if search != "" {
		query += " AND (title ILIKE $" + strconv.Itoa(len(args)+1) + " OR description ILIKE $" + strconv.Itoa(len(args)+2) + ")"
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

	var listings []Listing
	for rows.Next() {
		var l Listing
		var categoryID sql.NullString
		var avgRating sql.NullFloat64

		if err := rows.Scan(&l.ID, &l.ShopID, &categoryID, &l.Title, &l.Description, &l.Price, &l.Currency, &l.Quantity, &l.IsActive, &l.ViewsCount, &l.SalesCount, &avgRating, &l.CreatedAt, &l.UpdatedAt); err != nil {
			continue
		}

		if categoryID.Valid {
			l.CategoryID = categoryID.String
		}
		if avgRating.Valid {
			l.AvgRating = avgRating.Float64
		}

		// Load photos
		l.Photos = getListingPhotos(l.ID)
		listings = append(listings, l)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listings)
}

func handleGetListing(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var l Listing
	var categoryID sql.NullString
	var avgRating sql.NullFloat64

	err := db.QueryRow(
		"SELECT id, shop_id, category_id, title, description, price, currency, quantity, is_active, views_count, sales_count, avg_rating, created_at, updated_at FROM listings WHERE id = $1",
		id,
	).Scan(&l.ID, &l.ShopID, &categoryID, &l.Title, &l.Description, &l.Price, &l.Currency, &l.Quantity, &l.IsActive, &l.ViewsCount, &l.SalesCount, &avgRating, &l.CreatedAt, &l.UpdatedAt)

	if err != nil {
		http.Error(w, "Listing not found", http.StatusNotFound)
		return
	}

	if categoryID.Valid {
		l.CategoryID = categoryID.String
	}
	if avgRating.Valid {
		l.AvgRating = avgRating.Float64
	}

	// Increment views
	db.Exec("UPDATE listings SET views_count = views_count + 1 WHERE id = $1", id)

	l.Photos = getListingPhotos(id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(l)
}

func handleCreateListing(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")

	// Get user's shop
	var shopID string
	err := db.QueryRow("SELECT id FROM shops WHERE owner_id = $1", userID).Scan(&shopID)
	if err != nil {
		http.Error(w, "No shop found. Create a shop first", http.StatusForbidden)
		return
	}

	var req struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Price       float64  `json:"price"`
		CategoryID  string   `json:"category_id"`
		Quantity    int      `json:"quantity"`
		Photos      []string `json:"photos"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	listingID := uuid.New().String()

	_, err = db.Exec(
		"INSERT INTO listings (id, shop_id, category_id, title, description, price, quantity) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		listingID, shopID, req.CategoryID, req.Title, req.Description, req.Price, req.Quantity,
	)

	if err != nil {
		http.Error(w, "Failed to create listing", http.StatusInternalServerError)
		return
	}

	// Add photos
	for i, photoURL := range req.Photos {
		photoID := uuid.New().String()
		isPrimary := i == 0

		db.Exec(
			"INSERT INTO listing_photos (id, listing_id, url, is_primary, sort_order) VALUES ($1, $2, $3, $4, $5)",
			photoID, listingID, photoURL, isPrimary, i,
		)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": listingID, "message": "Listing created"})
}

func handleUpdateListing(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	listingID := r.PathValue("id")

	// Verify ownership
	var shopID string
	err := db.QueryRow("SELECT shop_id FROM listings WHERE id = $1", listingID).Scan(&shopID)
	if err != nil {
		http.Error(w, "Listing not found", http.StatusNotFound)
		return
	}

	var ownerID string
	db.QueryRow("SELECT owner_id FROM shops WHERE id = $1", shopID).Scan(&ownerID)
	if ownerID != userID {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Price       float64 `json:"price"`
		Quantity    int    `json:"quantity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(
		"UPDATE listings SET title = $1, description = $2, price = $3, quantity = $4, updated_at = NOW() WHERE id = $5",
		req.Title, req.Description, req.Price, req.Quantity, listingID,
	)

	if err != nil {
		http.Error(w, "Failed to update listing", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Listing updated"})
}

func handleDeleteListing(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	listingID := r.PathValue("id")

	// Verify ownership
	var shopID string
	err := db.QueryRow("SELECT shop_id FROM listings WHERE id = $1", listingID).Scan(&shopID)
	if err != nil {
		http.Error(w, "Listing not found", http.StatusNotFound)
		return
	}

	var ownerID string
	db.QueryRow("SELECT owner_id FROM shops WHERE id = $1", shopID).Scan(&ownerID)
	if ownerID != userID {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	_, err = db.Exec("DELETE FROM listings WHERE id = $1", listingID)
	if err != nil {
		http.Error(w, "Failed to delete listing", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Listing deleted"})
}

func getListingPhotos(listingID string) []Photo {
	rows, err := db.Query(
		"SELECT id, url, alt_text, is_primary FROM listing_photos WHERE listing_id = $1 ORDER BY sort_order",
		listingID,
	)
	if err != nil {
		log.Println("Error fetching photos:", err)
		return []Photo{}
	}
	defer rows.Close()

	var photos []Photo
	for rows.Next() {
		var p Photo
		var altText sql.NullString
		if err := rows.Scan(&p.ID, &p.URL, &altText, &p.IsPrimary); err != nil {
			continue
		}
		if altText.Valid {
			p.AltText = altText.String
		}
		photos = append(photos, p)
	}

	return photos
}
