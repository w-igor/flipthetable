package main

import (
	"encoding/json"
	"net/http"
)

type Category struct {
	ID          string `json:"id"`
	ParentID    string `json:"parent_id,omitempty"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
}

func handleGetCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, parent_id, name, slug, description FROM categories ORDER BY sort_order")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		var parentID interface{}

		if err := rows.Scan(&c.ID, &parentID, &c.Name, &c.Slug, &c.Description); err != nil {
			continue
		}

		if parentID != nil {
			c.ParentID = parentID.(string)
		}

		categories = append(categories, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}
