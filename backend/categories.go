package main

import (
	"context"
	"net/http"
	"time"
)

// handleGetCategories returns all product categories in sort order.
// Used for populating category filters on the product listing page.
func handleGetCategories(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Fetch all categories ordered by display priority
	rows, err := dbPool.Query(ctx, `
		SELECT id, parent_id, name, slug, description, sort_order
		FROM categories
		ORDER BY sort_order, name
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać kategorii")
		return
	}
	defer rows.Close()

	// Scan all categories into a slice
	categories := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.Description, &c.SortOrder); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu kategorii")
			return
		}
		categories = append(categories, c)
	}

	writeJSON(w, http.StatusOK, categories)
}
