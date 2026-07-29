package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

// A buyer can only review a purchase once the shop has actually confirmed/handled it.
var reviewableOrderStatuses = map[string]bool{
	"paid":       true,
	"processing": true,
	"shipped":    true,
	"delivered":  true,
}

func handleCreateReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	var req CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		writeError(w, http.StatusBadRequest, "Ocena musi być w zakresie 1-5")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var listingID, buyerID, orderStatus string
	err := dbPool.QueryRow(ctx, `
		SELECT oi.listing_id, o.buyer_id, o.status
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.id = $1
	`, req.OrderItemID).Scan(&listingID, &buyerID, &orderStatus)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "Pozycja zamówienia nie znaleziona")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}
	if buyerID != userID {
		writeError(w, http.StatusForbidden, "To nie jest Twoje zamówienie")
		return
	}
	if !reviewableOrderStatuses[orderStatus] {
		writeError(w, http.StatusBadRequest, "Nie możesz jeszcze ocenić tego zakupu")
		return
	}

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd bazy danych")
		return
	}
	defer tx.Rollback(ctx)

	var review Review
	err = tx.QueryRow(ctx, `
		INSERT INTO reviews (listing_id, reviewer_id, order_item_id, rating, comment)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		RETURNING id, listing_id, reviewer_id, order_item_id, rating, comment, created_at
	`, listingID, userID, req.OrderItemID, req.Rating, req.Comment).Scan(
		&review.ID, &review.ListingID, &review.ReviewerID, &review.OrderItemID,
		&review.Rating, &review.Comment, &review.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "Już oceniłeś ten zakup")
			return
		}
		writeError(w, http.StatusInternalServerError, "Nie udało się dodać opinii")
		return
	}

	if _, err := tx.Exec(ctx, `
		UPDATE listings SET avg_rating = (SELECT ROUND(AVG(rating), 2) FROM reviews WHERE listing_id = $1)
		WHERE id = $1
	`, listingID); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować oceny produktu")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać opinii")
		return
	}

	if err := dbPool.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, userID).Scan(&review.ReviewerUsername); err != nil {
		review.ReviewerUsername = ""
	}

	writeJSON(w, http.StatusCreated, review)
}

func handleGetListingReviews(w http.ResponseWriter, r *http.Request) {
	listingID := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := dbPool.Query(ctx, `
		SELECT r.id, r.listing_id, r.reviewer_id, u.username, r.order_item_id, r.rating, r.comment, r.created_at
		FROM reviews r
		JOIN users u ON u.id = r.reviewer_id
		WHERE r.listing_id = $1
		ORDER BY r.created_at DESC
		LIMIT 50
	`, listingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać opinii")
		return
	}
	defer rows.Close()

	reviews := []Review{}
	for rows.Next() {
		var rv Review
		if err := rows.Scan(&rv.ID, &rv.ListingID, &rv.ReviewerID, &rv.ReviewerUsername, &rv.OrderItemID, &rv.Rating, &rv.Comment, &rv.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu opinii")
			return
		}
		reviews = append(reviews, rv)
	}

	writeJSON(w, http.StatusOK, reviews)
}
