package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func handleGetListings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 24
	}

	conditions := []string{"l.is_active = TRUE"}
	args := []interface{}{}
	argN := 0

	nextArg := func(v interface{}) string {
		argN++
		args = append(args, v)
		return fmt.Sprintf("$%d", argN)
	}

	if categorySlug := strings.TrimSpace(q.Get("category")); categorySlug != "" {
		conditions = append(conditions, "c.slug = "+nextArg(categorySlug))
	}
	if search := strings.TrimSpace(q.Get("q")); search != "" {
		placeholder := nextArg("%" + search + "%")
		conditions = append(conditions, "(l.title ILIKE "+placeholder+" OR l.description ILIKE "+placeholder+")")
	}
	if minPrice := strings.TrimSpace(q.Get("min_price")); minPrice != "" {
		if v, err := strconv.ParseFloat(minPrice, 64); err == nil {
			conditions = append(conditions, "l.price >= "+nextArg(v))
		}
	}
	if maxPrice := strings.TrimSpace(q.Get("max_price")); maxPrice != "" {
		if v, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			conditions = append(conditions, "l.price <= "+nextArg(v))
		}
	}
	if q.Get("in_stock") == "true" {
		conditions = append(conditions, "l.quantity > 0")
	}

	whereClause := strings.Join(conditions, " AND ")

	orderBy := "l.created_at DESC"
	switch q.Get("sort") {
	case "price_asc":
		orderBy = "l.price ASC"
	case "price_desc":
		orderBy = "l.price DESC"
	case "popular":
		orderBy = "l.sales_count DESC"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	countQuery := `
		SELECT COUNT(*)
		FROM listings l
		JOIN shops s ON s.id = l.shop_id
		LEFT JOIN categories c ON c.id = l.category_id
		WHERE ` + whereClause

	var total int
	if err := dbPool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się policzyć produktów")
		return
	}

	limitArg := nextArg(pageSize)
	offsetArg := nextArg((page - 1) * pageSize)

	listQuery := `
		SELECT
			l.id, l.shop_id, s.name, l.category_id, l.title, l.description,
			l.price, l.currency, l.quantity, l.is_active, l.views_count,
			l.sales_count, l.avg_rating, l.created_at,
			(SELECT url FROM listing_photos p WHERE p.listing_id = l.id ORDER BY p.is_primary DESC, p.sort_order LIMIT 1)
		FROM listings l
		JOIN shops s ON s.id = l.shop_id
		LEFT JOIN categories c ON c.id = l.category_id
		WHERE ` + whereClause + `
		ORDER BY ` + orderBy + `
		LIMIT ` + limitArg + ` OFFSET ` + offsetArg

	rows, err := dbPool.Query(ctx, listQuery, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać produktów")
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
			writeError(w, http.StatusInternalServerError, "Błąd odczytu produktów")
			return
		}
		items = append(items, l)
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	writeJSON(w, http.StatusOK, ListingsResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	})
}

func handleGetListing(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var l Listing
	err := dbPool.QueryRow(ctx, `
		SELECT
			l.id, l.shop_id, s.name, l.category_id, l.title, l.description,
			l.price, l.currency, l.quantity, l.is_active, l.views_count,
			l.sales_count, l.avg_rating, l.created_at
		FROM listings l
		JOIN shops s ON s.id = l.shop_id
		WHERE l.id = $1 AND l.is_active = TRUE
	`, id).Scan(
		&l.ID, &l.ShopID, &l.ShopName, &l.CategoryID, &l.Title, &l.Description,
		&l.Price, &l.Currency, &l.Quantity, &l.IsActive, &l.ViewsCount,
		&l.SalesCount, &l.AvgRating, &l.CreatedAt,
	)
	if err != nil {
		writeError(w, http.StatusNotFound, "Produkt nie znaleziony")
		return
	}

	rows, err := dbPool.Query(ctx, `
		SELECT id, url, alt_text, is_primary, sort_order
		FROM listing_photos
		WHERE listing_id = $1
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

	go dbPool.Exec(context.Background(), `UPDATE listings SET views_count = views_count + 1 WHERE id = $1`, id)

	writeJSON(w, http.StatusOK, l)
}
