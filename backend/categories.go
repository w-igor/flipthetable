package main

import (
	"context"
	"net/http"
	"time"
)

func handleGetCategories(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

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
