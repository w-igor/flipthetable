package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var polishDiacritics = strings.NewReplacer(
	"ą", "a", "ć", "c", "ę", "e", "ł", "l", "ń", "n",
	"ó", "o", "ś", "s", "ź", "z", "ż", "z",
)

func slugify(name string) string {
	s := polishDiacritics.Replace(strings.ToLower(name))

	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// getOwnShopID looks up the shop owned by the current user. Returns pgx.ErrNoRows if they don't have one yet.
func getOwnShopID(ctx context.Context, userID string) (string, error) {
	var shopID string
	err := dbPool.QueryRow(ctx, `SELECT id FROM shops WHERE owner_id = $1`, userID).Scan(&shopID)
	return shopID, err
}

func handleCreateShop(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	var req ShopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 3 {
		writeError(w, http.StatusBadRequest, "Nazwa sklepu musi mieć min. 3 znaki")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if _, err := getOwnShopID(ctx, userID); err == nil {
		writeError(w, http.StatusConflict, "Masz już sklep")
		return
	}

	baseSlug := slugify(req.Name)
	if baseSlug == "" {
		baseSlug = "sklep"
	}

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd bazy danych")
		return
	}
	defer tx.Rollback(ctx)

	var shop Shop
	for attempt := 0; attempt < 5; attempt++ {
		slug := baseSlug
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%d", baseSlug, attempt+1)
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO shops (owner_id, name, slug, description, banner_url, avatar_url)
			VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''))
			RETURNING id, owner_id, name, slug, description, banner_url, avatar_url, is_active, sales_count, created_at
		`, userID, req.Name, slug, req.Description, req.BannerURL, req.AvatarURL).Scan(
			&shop.ID, &shop.OwnerID, &shop.Name, &shop.Slug, &shop.Description,
			&shop.BannerURL, &shop.AvatarURL, &shop.IsActive, &shop.SalesCount, &shop.CreatedAt,
		)
		if err == nil {
			break
		}
		if !isUniqueViolation(err) {
			writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć sklepu")
			return
		}
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć sklepu")
		return
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET is_seller = TRUE WHERE id = $1`, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować konta")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się sfinalizować tworzenia sklepu")
		return
	}

	writeJSON(w, http.StatusCreated, shop)
}

func handleGetMyShop(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var shop Shop
	err := dbPool.QueryRow(ctx, `
		SELECT id, owner_id, name, slug, description, banner_url, avatar_url, is_active, sales_count, created_at
		FROM shops WHERE owner_id = $1
	`, userID).Scan(
		&shop.ID, &shop.OwnerID, &shop.Name, &shop.Slug, &shop.Description,
		&shop.BannerURL, &shop.AvatarURL, &shop.IsActive, &shop.SalesCount, &shop.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "Nie masz jeszcze sklepu")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}

	writeJSON(w, http.StatusOK, shop)
}

func handleUpdateShop(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	var req ShopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 3 {
		writeError(w, http.StatusBadRequest, "Nazwa sklepu musi mieć min. 3 znaki")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var shop Shop
	err := dbPool.QueryRow(ctx, `
		UPDATE shops
		SET name = $1, description = NULLIF($2, ''), banner_url = NULLIF($3, ''), avatar_url = NULLIF($4, '')
		WHERE owner_id = $5
		RETURNING id, owner_id, name, slug, description, banner_url, avatar_url, is_active, sales_count, created_at
	`, req.Name, req.Description, req.BannerURL, req.AvatarURL, userID).Scan(
		&shop.ID, &shop.OwnerID, &shop.Name, &shop.Slug, &shop.Description,
		&shop.BannerURL, &shop.AvatarURL, &shop.IsActive, &shop.SalesCount, &shop.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "Nie masz jeszcze sklepu")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować sklepu")
		return
	}

	writeJSON(w, http.StatusOK, shop)
}

func handleGetShopPublic(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var shop ShopProfile
	err := dbPool.QueryRow(ctx, `
		SELECT s.id, s.owner_id, s.name, s.slug, s.description, s.banner_url, s.avatar_url,
		       s.is_active, s.sales_count, s.created_at,
		       (SELECT COUNT(*) FROM listings l WHERE l.shop_id = s.id AND l.is_active = TRUE)
		FROM shops s
		WHERE s.id = $1 AND s.is_active = TRUE
	`, id).Scan(
		&shop.ID, &shop.OwnerID, &shop.Name, &shop.Slug, &shop.Description,
		&shop.BannerURL, &shop.AvatarURL, &shop.IsActive, &shop.SalesCount, &shop.CreatedAt,
		&shop.ListingsCount,
	)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "Sklep nie znaleziony")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}

	writeJSON(w, http.StatusOK, shop)
}

func handleGetSellerStats(w http.ResponseWriter, r *http.Request) {
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

	stats := SellerStats{OrdersByStatus: map[string]int{}}

	var totalRevenue float64
	err = dbPool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total_amount) FILTER (WHERE status NOT IN ('cancelled', 'refunded')), 0)
		FROM orders WHERE shop_id = $1
	`, shopID).Scan(&stats.TotalOrders, &totalRevenue)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać statystyk")
		return
	}
	stats.TotalRevenue = formatPrice(totalRevenue)

	statusRows, err := dbPool.Query(ctx, `SELECT status, COUNT(*) FROM orders WHERE shop_id = $1 GROUP BY status`, shopID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać statystyk zamówień")
		return
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu statystyk")
			return
		}
		stats.OrdersByStatus[status] = count
	}

	err = dbPool.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE is_active), COUNT(*)
		FROM listings WHERE shop_id = $1
	`, shopID).Scan(&stats.ListingsActive, &stats.ListingsTotal)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać statystyk ofert")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func handleGetMyListings(w http.ResponseWriter, r *http.Request) {
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
		SELECT
			l.id, l.shop_id, s.name, l.category_id, l.title, l.description,
			l.price, l.currency, l.quantity, l.is_active, l.has_variants, l.shipping_profile_id, l.views_count,
			l.sales_count, l.avg_rating, l.created_at,
			(SELECT url FROM listing_photos p WHERE p.listing_id = l.id ORDER BY p.is_primary DESC, p.sort_order LIMIT 1)
		FROM listings l
		JOIN shops s ON s.id = l.shop_id
		WHERE l.shop_id = $1
		ORDER BY l.created_at DESC
		LIMIT 200
	`, shopID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać ofert")
		return
	}
	defer rows.Close()

	items := []Listing{}
	for rows.Next() {
		var l Listing
		if err := rows.Scan(
			&l.ID, &l.ShopID, &l.ShopName, &l.CategoryID, &l.Title, &l.Description,
			&l.Price, &l.Currency, &l.Quantity, &l.IsActive, &l.HasVariants, &l.ShippingProfileID, &l.ViewsCount,
			&l.SalesCount, &l.AvgRating, &l.CreatedAt, &l.PrimaryPhoto,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu ofert")
			return
		}
		items = append(items, l)
	}
	for i := range items {
		attachShipping(ctx, &items[i])
	}

	writeJSON(w, http.StatusOK, items)
}

func handleGetMyListingByID(w http.ResponseWriter, r *http.Request) {
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
		writeError(w, http.StatusNotFound, "Nie masz jeszcze sklepu")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}

	var owns bool
	if err := dbPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM listings WHERE id = $1 AND shop_id = $2)`, id, shopID).Scan(&owns); err != nil || !owns {
		writeError(w, http.StatusNotFound, "Oferta nie znaleziona w Twoim sklepie")
		return
	}

	listing, err := fetchListingByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać oferty")
		return
	}

	writeJSON(w, http.StatusOK, listing)
}

func validateListingRequest(req *ListingRequest) string {
	req.Title = strings.TrimSpace(req.Title)
	if len(req.Title) < 3 {
		return "Tytuł oferty musi mieć min. 3 znaki"
	}
	price, err := strconv.ParseFloat(req.Price, 64)
	if err != nil || price < 0 {
		return "Podaj prawidłową cenę"
	}
	if req.Quantity < 0 {
		return "Ilość nie może być ujemna"
	}
	if req.Currency == "" {
		req.Currency = "PLN"
	}
	return ""
}

func replaceListingPhotos(ctx context.Context, tx pgx.Tx, listingID string, photos []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM listing_photos WHERE listing_id = $1`, listingID); err != nil {
		return err
	}
	for i, url := range photos {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO listing_photos (listing_id, url, is_primary, sort_order)
			VALUES ($1, $2, $3, $4)
		`, listingID, url, i == 0, i); err != nil {
			return err
		}
	}
	return nil
}

func fetchListingByID(ctx context.Context, id string) (Listing, error) {
	var l Listing
	err := dbPool.QueryRow(ctx, `
		SELECT l.id, l.shop_id, s.name, l.category_id, l.title, l.description,
		       l.price, l.currency, l.quantity, l.is_active, l.has_variants, l.shipping_profile_id, l.views_count,
		       l.sales_count, l.avg_rating, l.created_at
		FROM listings l
		JOIN shops s ON s.id = l.shop_id
		WHERE l.id = $1
	`, id).Scan(
		&l.ID, &l.ShopID, &l.ShopName, &l.CategoryID, &l.Title, &l.Description,
		&l.Price, &l.Currency, &l.Quantity, &l.IsActive, &l.HasVariants, &l.ShippingProfileID, &l.ViewsCount,
		&l.SalesCount, &l.AvgRating, &l.CreatedAt,
	)
	if err != nil {
		return l, err
	}
	attachShipping(ctx, &l)

	rows, err := dbPool.Query(ctx, `
		SELECT id, url, alt_text, is_primary, sort_order
		FROM listing_photos WHERE listing_id = $1
		ORDER BY is_primary DESC, sort_order
	`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p ListingPhoto
			if rows.Scan(&p.ID, &p.URL, &p.AltText, &p.IsPrimary, &p.SortOrder) == nil {
				l.Photos = append(l.Photos, p)
			}
		}
	}

	if l.HasVariants {
		l.VariantTypes, l.VariantSkus = fetchListingVariants(ctx, id)
	}

	return l, nil
}

func handleCreateListing(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	var req ListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	if msg := validateListingRequest(&req); msg != "" {
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

	if msg := validateOwnShippingProfile(ctx, shopID, &req.ShippingProfileID); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd bazy danych")
		return
	}
	defer tx.Rollback(ctx)

	var listingID string
	err = tx.QueryRow(ctx, `
		INSERT INTO listings (shop_id, category_id, title, description, price, currency, quantity, shipping_profile_id)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8)
		RETURNING id
	`, shopID, req.CategoryID, req.Title, req.Description, req.Price, req.Currency, req.Quantity, req.ShippingProfileID).Scan(&listingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć oferty")
		return
	}

	if err := replaceListingPhotos(ctx, tx, listingID, req.Photos); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać zdjęć")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się sfinalizować oferty")
		return
	}

	listing, err := fetchListingByID(context.Background(), listingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Oferta utworzona, ale nie udało się jej odczytać")
		return
	}

	writeJSON(w, http.StatusCreated, listing)
}

func handleUpdateListing(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}
	id := r.PathValue("id")

	var req ListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	if msg := validateListingRequest(&req); msg != "" {
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

	if msg := validateOwnShippingProfile(ctx, shopID, &req.ShippingProfileID); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd bazy danych")
		return
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE listings
		SET category_id = $1, title = $2, description = NULLIF($3, ''), price = $4,
		    currency = $5, quantity = $6, is_active = $7, shipping_profile_id = $8
		WHERE id = $9 AND shop_id = $10
	`, req.CategoryID, req.Title, req.Description, req.Price, req.Currency, req.Quantity, isActive, req.ShippingProfileID, id, shopID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować oferty")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Oferta nie znaleziona w Twoim sklepie")
		return
	}

	if req.Photos != nil {
		if err := replaceListingPhotos(ctx, tx, id, req.Photos); err != nil {
			writeError(w, http.StatusInternalServerError, "Nie udało się zapisać zdjęć")
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się sfinalizować zmian")
		return
	}

	listing, err := fetchListingByID(context.Background(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Oferta zaktualizowana, ale nie udało się jej odczytać")
		return
	}

	writeJSON(w, http.StatusOK, listing)
}

func handleDeleteListing(w http.ResponseWriter, r *http.Request) {
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

	tag, err := dbPool.Exec(ctx, `
		UPDATE listings SET is_active = FALSE WHERE id = $1 AND shop_id = $2
	`, id, shopID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się usunąć oferty")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Oferta nie znaleziona w Twoim sklepie")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Oferta usunięta"})
}
