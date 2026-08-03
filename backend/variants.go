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

// fetchListingVariants loads the variant types (e.g., Size, Color) with their options (e.g., S, M, L)
// and all SKU (Stock Keeping Unit) combinations for a listing with their prices and quantities.
// Uses best-effort error handling: on query failure returns empty slices rather than failing the response.
func fetchListingVariants(ctx context.Context, listingID string) ([]VariantType, []VariantSku) {
	types := []VariantType{}
	typeRows, err := dbPool.Query(ctx, `
		SELECT id, name, position FROM listing_variant_types
		WHERE listing_id = $1 ORDER BY position
	`, listingID)
	if err == nil {
		defer typeRows.Close()
		for typeRows.Next() {
			var t VariantType
			if typeRows.Scan(&t.ID, &t.Name, &t.Position) == nil {
				types = append(types, t)
			}
		}
	}

	for i := range types {
		optRows, err := dbPool.Query(ctx, `
			SELECT id, value, sort_order FROM listing_variant_options
			WHERE variant_type_id = $1 ORDER BY sort_order
		`, types[i].ID)
		if err != nil {
			continue
		}
		for optRows.Next() {
			var o VariantOption
			if optRows.Scan(&o.ID, &o.Value, &o.SortOrder) == nil {
				types[i].Options = append(types[i].Options, o)
			}
		}
		optRows.Close()
	}

	skus := []VariantSku{}
	skuRows, err := dbPool.Query(ctx, `
		SELECT s.id, s.option1_id, s.option2_id, s.price, s.quantity,
		       t1.name, o1.value, t2.name, o2.value
		FROM listing_variant_skus s
		LEFT JOIN listing_variant_options o1 ON o1.id = s.option1_id
		LEFT JOIN listing_variant_types t1 ON t1.id = o1.variant_type_id
		LEFT JOIN listing_variant_options o2 ON o2.id = s.option2_id
		LEFT JOIN listing_variant_types t2 ON t2.id = o2.variant_type_id
		WHERE s.listing_id = $1
	`, listingID)
	if err == nil {
		defer skuRows.Close()
		for skuRows.Next() {
			var s VariantSku
			var t1Name, o1Value, t2Name, o2Value *string
			if skuRows.Scan(&s.ID, &s.Option1ID, &s.Option2ID, &s.Price, &s.Quantity, &t1Name, &o1Value, &t2Name, &o2Value) != nil {
				continue
			}
			s.Label = variantLabel(t1Name, o1Value, t2Name, o2Value)
			skus = append(skus, s)
		}
	}

	return types, skus
}

// variantLabel builds a human-readable label from variant type names and option values.
// E.g., "Size: M, Color: Red" from the two variant dimensions.
func variantLabel(t1Name, o1Value, t2Name, o2Value *string) string {
	parts := []string{}
	if t1Name != nil && o1Value != nil {
		parts = append(parts, fmt.Sprintf("%s: %s", *t1Name, *o1Value))
	}
	if t2Name != nil && o2Value != nil {
		parts = append(parts, fmt.Sprintf("%s: %s", *t2Name, *o2Value))
	}
	return strings.Join(parts, ", ")
}

// handleReplaceListingVariants replaces all variants (size/color combos) for a listing.
// Validates variant structure, removes duplicates, and ensures seller owns the listing.
// Supports up to 2 variant types with multiple options each, generating SKUs (combinations) with prices.
func handleReplaceListingVariants(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}
	listingID := r.PathValue("id")

	var req ReplaceVariantsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	// Validate: max 2 variant types (e.g., Size and Color)
	if len(req.Types) > 2 {
		writeError(w, http.StatusBadRequest, "Można dodać maksymalnie 2 typy wariacji")
		return
	}
	// Validate and clean variant types
	for i := range req.Types {
		req.Types[i].Name = strings.TrimSpace(req.Types[i].Name)
		if req.Types[i].Name == "" {
			writeError(w, http.StatusBadRequest, "Nazwa typu wariacji nie może być pusta")
			return
		}
		// Remove duplicate and empty option values
		seen := map[string]bool{}
		cleaned := make([]string, 0, len(req.Types[i].Options))
		for _, v := range req.Types[i].Options {
			v = strings.TrimSpace(v)
			if v == "" || seen[v] {
				continue
			}
			seen[v] = true
			cleaned = append(cleaned, v)
		}
		if len(cleaned) == 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Typ wariacji \"%s\" musi mieć co najmniej jedną wartość", req.Types[i].Name))
			return
		}
		req.Types[i].Options = cleaned
	}
	// Validate SKU combinations
	for i := range req.Skus {
		// Each SKU must have one value per variant type
		if len(req.Skus[i].OptionValues) != len(req.Types) {
			writeError(w, http.StatusBadRequest, "Każda kombinacja musi mieć wartość dla każdego typu wariacji")
			return
		}
		if req.Skus[i].Quantity < 0 {
			writeError(w, http.StatusBadRequest, "Ilość nie może być ujemna")
			return
		}
		// Validate price format if provided (variant can override base listing price)
		if req.Skus[i].Price != nil {
			if _, err := strconv.ParseFloat(*req.Skus[i].Price, 64); err != nil {
				writeError(w, http.StatusBadRequest, "Nieprawidłowa cena wariantu")
				return
			}
		}
	}
	// Prevent adding SKUs without any variant types
	if len(req.Types) == 0 && len(req.Skus) > 0 {
		writeError(w, http.StatusBadRequest, "Nie można dodać kombinacji bez typów wariacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
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

	var owns bool
	if err := dbPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM listings WHERE id = $1 AND shop_id = $2)`, listingID, shopID).Scan(&owns); err != nil || !owns {
		writeError(w, http.StatusNotFound, "Oferta nie znaleziona w Twoim sklepie")
		return
	}

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd bazy danych")
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM listing_variant_types WHERE listing_id = $1`, listingID); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się wyczyścić poprzednich wariantów")
		return
	}

	// optionIDsByPosition[position][value] = option UUID, used to resolve sku option_values into IDs.
	optionIDsByPosition := make([]map[string]string, len(req.Types))
	for i, t := range req.Types {
		var typeID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO listing_variant_types (listing_id, name, position) VALUES ($1, $2, $3) RETURNING id
		`, listingID, t.Name, i+1).Scan(&typeID); err != nil {
			writeError(w, http.StatusInternalServerError, "Nie udało się zapisać typu wariacji")
			return
		}
		optionIDsByPosition[i] = map[string]string{}
		for j, v := range t.Options {
			var optID string
			if err := tx.QueryRow(ctx, `
				INSERT INTO listing_variant_options (variant_type_id, value, sort_order) VALUES ($1, $2, $3) RETURNING id
			`, typeID, v, j).Scan(&optID); err != nil {
				writeError(w, http.StatusInternalServerError, "Nie udało się zapisać wartości wariacji")
				return
			}
			optionIDsByPosition[i][v] = optID
		}
	}

	totalQuantity := 0
	for _, sReq := range req.Skus {
		var opt1, opt2 *string
		for i, v := range sReq.OptionValues {
			id, found := optionIDsByPosition[i][strings.TrimSpace(v)]
			if !found {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Nieznana wartość wariacji \"%s\"", v))
				return
			}
			if i == 0 {
				opt1 = &id
			} else {
				opt2 = &id
			}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO listing_variant_skus (listing_id, option1_id, option2_id, price, quantity)
			VALUES ($1, $2, $3, $4, $5)
		`, listingID, opt1, opt2, sReq.Price, sReq.Quantity); err != nil {
			if isUniqueViolation(err) {
				writeError(w, http.StatusBadRequest, "Ta kombinacja wariantów została podana więcej niż raz")
				return
			}
			writeError(w, http.StatusInternalServerError, "Nie udało się zapisać kombinacji wariantu")
			return
		}
		totalQuantity += sReq.Quantity
	}

	hasVariants := len(req.Types) > 0
	if hasVariants {
		if _, err := tx.Exec(ctx, `UPDATE listings SET has_variants = TRUE, quantity = $1 WHERE id = $2`, totalQuantity, listingID); err != nil {
			writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować oferty")
			return
		}
	} else {
		if _, err := tx.Exec(ctx, `UPDATE listings SET has_variants = FALSE WHERE id = $1`, listingID); err != nil {
			writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować oferty")
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się sfinalizować zmian")
		return
	}

	listing, err := fetchListingByID(context.Background(), listingID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Warianty zapisane, ale nie udało się odczytać oferty")
		return
	}

	writeJSON(w, http.StatusOK, listing)
}
