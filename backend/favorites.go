package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type FavoriteRequest struct {
	ListingID string `json:"listing_id"`
}

func handleAddFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	var req FavoriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ListingID == "" {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err := dbPool.Exec(ctx, `
		INSERT INTO favorites (user_id, listing_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, req.ListingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się dodać do ulubionych")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "Dodano do ulubionych"})
}

func handleRemoveFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}
	listingID := r.PathValue("listingId")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := dbPool.Exec(ctx, `DELETE FROM favorites WHERE user_id = $1 AND listing_id = $2`, userID, listingID); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się usunąć z ulubionych")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Usunięto z ulubionych"})
}

func handleListFavorites(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := dbPool.Query(ctx, `
		SELECT
			l.id, l.shop_id, s.name, l.category_id, l.title, l.description,
			l.price, l.currency, l.quantity, l.is_active, l.views_count,
			l.sales_count, l.avg_rating, l.created_at,
			(SELECT url FROM listing_photos p WHERE p.listing_id = l.id ORDER BY p.is_primary DESC, p.sort_order LIMIT 1)
		FROM favorites f
		JOIN listings l ON l.id = f.listing_id
		JOIN shops s ON s.id = l.shop_id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC
	`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać ulubionych")
		return
	}
	defer rows.Close()

	items := []Listing{}
	for rows.Next() {
		var l Listing
		if err := rows.Scan(
			&l.ID, &l.ShopID, &l.ShopName, &l.CategoryID, &l.Title, &l.Description,
			&l.Price, &l.Currency, &l.Quantity, &l.IsActive, &l.ViewsCount,
			&l.SalesCount, &l.AvgRating, &l.CreatedAt, &l.PrimaryPhoto,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu ulubionych")
			return
		}
		items = append(items, l)
	}

	writeJSON(w, http.StatusOK, items)
}

func handleListFavoriteIDs(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := dbPool.Query(ctx, `SELECT listing_id FROM favorites WHERE user_id = $1`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać ulubionych")
		return
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}

	writeJSON(w, http.StatusOK, ids)
}
