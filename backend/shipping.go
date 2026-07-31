package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func validateShippingProfileRequest(req *ShippingProfileRequest) string {
	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 2 {
		return "Nazwa profilu musi mieć min. 2 znaki"
	}
	if _, err := strconv.ParseFloat(req.Price, 64); err != nil {
		return "Podaj prawidłową cenę wysyłki"
	}
	if req.MinDays < 0 || req.MaxDays < req.MinDays {
		return "Nieprawidłowy zakres dni dostawy"
	}
	return ""
}

// validateOwnShippingProfile normalizes an empty-string ID to nil and, when set,
// checks the profile belongs to the seller's own shop. Returns a user-facing
// error message, or "" if valid.
func validateOwnShippingProfile(ctx context.Context, shopID string, id **string) string {
	if *id != nil && **id == "" {
		*id = nil
	}
	if *id == nil {
		return ""
	}
	var exists bool
	if err := dbPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM shipping_profiles WHERE id = $1 AND shop_id = $2)`, **id, shopID).Scan(&exists); err != nil || !exists {
		return "Wybrany profil wysyłki nie istnieje"
	}
	return ""
}

// attachShipping loads the shipping profile referenced by the listing (if any) and embeds it.
func attachShipping(ctx context.Context, l *Listing) {
	if l.ShippingProfileID == nil {
		return
	}
	var sp ShippingProfile
	err := dbPool.QueryRow(ctx, `
		SELECT id, shop_id, name, price, min_days, max_days, created_at
		FROM shipping_profiles WHERE id = $1
	`, *l.ShippingProfileID).Scan(&sp.ID, &sp.ShopID, &sp.Name, &sp.Price, &sp.MinDays, &sp.MaxDays, &sp.CreatedAt)
	if err == nil {
		l.Shipping = &sp
	}
}

func handleListShippingProfiles(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	shopID, err := getOwnShopID(ctx, userID)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "Nie masz jeszcze sklepu")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}

	rows, err := dbPool.Query(ctx, `
		SELECT id, shop_id, name, price, min_days, max_days, created_at
		FROM shipping_profiles WHERE shop_id = $1
		ORDER BY created_at
	`, shopID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać profili wysyłki")
		return
	}
	defer rows.Close()

	profiles := []ShippingProfile{}
	for rows.Next() {
		var sp ShippingProfile
		if err := rows.Scan(&sp.ID, &sp.ShopID, &sp.Name, &sp.Price, &sp.MinDays, &sp.MaxDays, &sp.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu profili wysyłki")
			return
		}
		profiles = append(profiles, sp)
	}

	writeJSON(w, http.StatusOK, profiles)
}

func handleCreateShippingProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	var req ShippingProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	if msg := validateShippingProfileRequest(&req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	shopID, err := getOwnShopID(ctx, userID)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusBadRequest, "Najpierw załóż sklep")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}

	var sp ShippingProfile
	err = dbPool.QueryRow(ctx, `
		INSERT INTO shipping_profiles (shop_id, name, price, min_days, max_days)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, shop_id, name, price, min_days, max_days, created_at
	`, shopID, req.Name, req.Price, req.MinDays, req.MaxDays).Scan(
		&sp.ID, &sp.ShopID, &sp.Name, &sp.Price, &sp.MinDays, &sp.MaxDays, &sp.CreatedAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć profilu wysyłki")
		return
	}

	writeJSON(w, http.StatusCreated, sp)
}

func handleUpdateShippingProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}
	id := r.PathValue("id")

	var req ShippingProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	if msg := validateShippingProfileRequest(&req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	shopID, err := getOwnShopID(ctx, userID)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusBadRequest, "Najpierw załóż sklep")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}

	var sp ShippingProfile
	err = dbPool.QueryRow(ctx, `
		UPDATE shipping_profiles
		SET name = $1, price = $2, min_days = $3, max_days = $4
		WHERE id = $5 AND shop_id = $6
		RETURNING id, shop_id, name, price, min_days, max_days, created_at
	`, req.Name, req.Price, req.MinDays, req.MaxDays, id, shopID).Scan(
		&sp.ID, &sp.ShopID, &sp.Name, &sp.Price, &sp.MinDays, &sp.MaxDays, &sp.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "Profil wysyłki nie znaleziony")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować profilu wysyłki")
		return
	}

	writeJSON(w, http.StatusOK, sp)
}

func handleDeleteShippingProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}
	id := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	shopID, err := getOwnShopID(ctx, userID)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusBadRequest, "Najpierw załóż sklep")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}

	tag, err := dbPool.Exec(ctx, `DELETE FROM shipping_profiles WHERE id = $1 AND shop_id = $2`, id, shopID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się usunąć profilu wysyłki")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Profil wysyłki nie znaleziony")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Profil wysyłki usunięty"})
}
