package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	etsyAPIBase          = "https://openapi.etsy.com/v3/application"
	etsyMaxSelectedItems = 50
	etsyMaxAllPages      = 5 // safety cap: up to 5 * etsyPageSize listings per "import all" run
	etsyPageSize         = 100
)

var errEtsyNotConfigured = errors.New("Klucz API Etsy nie jest jeszcze skonfigurowany")

type etsyMoney struct {
	Amount       int    `json:"amount"`
	Divisor      int    `json:"divisor"`
	CurrencyCode string `json:"currency_code"`
}

type etsyImage struct {
	URLFullxfull string `json:"url_fullxfull"`
}

type etsyListing struct {
	ListingID   int64       `json:"listing_id"`
	ShopID      int64       `json:"shop_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Price       etsyMoney   `json:"price"`
	Quantity    int         `json:"quantity"`
	State       string      `json:"state"`
	Images      []etsyImage `json:"images"`
}

type etsyListingsPage struct {
	Count   int           `json:"count"`
	Results []etsyListing `json:"results"`
}

func etsyAPIKey() string {
	return strings.TrimSpace(os.Getenv("ETSY_API_KEY"))
}

func etsySharedSecret() string {
	return strings.TrimSpace(os.Getenv("ETSY_SHARED_SECRET"))
}

// etsyAuthHeader builds the x-api-key header value. Etsy requires the shared
// secret to be appended as "keystring:shared_secret" as of Feb 9 2026.
func etsyAuthHeader() string {
	apiKey := etsyAPIKey()
	if secret := etsySharedSecret(); secret != "" {
		return apiKey + ":" + secret
	}
	return apiKey
}

func etsyGet(ctx context.Context, path string, query url.Values, out interface{}) error {
	if etsyAPIKey() == "" {
		return errEtsyNotConfigured
	}

	fullURL := etsyAPIBase + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", etsyAuthHeader())

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("Etsy API zwróciło błąd (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func fetchEtsyActiveListings(ctx context.Context, etsyShopID string) ([]etsyListing, error) {
	var all []etsyListing
	for page := 0; page < etsyMaxAllPages; page++ {
		var result etsyListingsPage
		query := url.Values{
			"limit":    {strconv.Itoa(etsyPageSize)},
			"offset":   {strconv.Itoa(page * etsyPageSize)},
			"includes": {"Images"},
		}
		if err := etsyGet(ctx, "/shops/"+etsyShopID+"/listings/active", query, &result); err != nil {
			return nil, err
		}
		all = append(all, result.Results...)
		if len(result.Results) < etsyPageSize {
			break
		}
	}

	// The shop-level listing endpoint doesn't reliably embed images via
	// "includes=Images" the way the single-listing endpoint does, so
	// backfill them per-listing whenever they came back empty.
	for i := range all {
		if len(all[i].Images) > 0 {
			continue
		}
		images, err := fetchEtsyListingImages(ctx, strconv.FormatInt(all[i].ListingID, 10))
		if err == nil {
			all[i].Images = images
		}
	}

	return all, nil
}

func fetchEtsyListingImages(ctx context.Context, listingID string) ([]etsyImage, error) {
	var result struct {
		Results []etsyImage `json:"results"`
	}
	if err := etsyGet(ctx, "/listings/"+listingID+"/images", nil, &result); err != nil {
		return nil, err
	}
	return result.Results, nil
}

func fetchEtsyListingByID(ctx context.Context, listingID string) (etsyListing, error) {
	var l etsyListing
	err := etsyGet(ctx, "/listings/"+listingID, url.Values{"includes": {"Images"}}, &l)
	return l, err
}

type etsyImportRequest struct {
	Mode       string   `json:"mode"` // "all" | "selected"
	ListingIDs []string `json:"listing_ids"`
}

type etsyImportError struct {
	EtsyListingID string `json:"etsy_listing_id"`
	Message       string `json:"message"`
}

type etsyImportResult struct {
	Fetched  int               `json:"fetched"`
	Imported int               `json:"imported"`
	Skipped  int               `json:"skipped"`
	Errors   []etsyImportError `json:"errors"`
}

func handleEtsyStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := map[string]interface{}{
		"api_key_configured": etsyAPIKey() != "",
		"connected":          false,
		"etsy_shop_id":       nil,
	}

	shopID, err := getOwnShopID(ctx, userID)
	if err == nil {
		var etsyShopID string
		if err := dbPool.QueryRow(ctx, `SELECT etsy_shop_id FROM etsy_connections WHERE shop_id = $1`, shopID).Scan(&etsyShopID); err == nil {
			resp["connected"] = true
			resp["etsy_shop_id"] = etsyShopID
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func handleEtsyDisconnect(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	shopID, err := getOwnShopID(ctx, userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Najpierw załóż sklep")
		return
	}

	if _, err := dbPool.Exec(ctx, `DELETE FROM etsy_connections WHERE shop_id = $1`, shopID); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się rozłączyć konta Etsy")
		return
	}
	_, _ = dbPool.Exec(ctx, `UPDATE shops SET etsy_shop_id = NULL WHERE id = $1`, shopID)

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleEtsyImport(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	if etsyAPIKey() == "" {
		writeError(w, http.StatusServiceUnavailable, "Klucz API Etsy nie jest jeszcze skonfigurowany")
		return
	}

	var req etsyImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	req.Mode = strings.TrimSpace(req.Mode)
	if req.Mode != "all" && req.Mode != "selected" {
		writeError(w, http.StatusBadRequest, "Nieprawidłowy tryb importu")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
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

	var connectedEtsyShopID string
	err = dbPool.QueryRow(ctx, `SELECT etsy_shop_id FROM etsy_connections WHERE shop_id = $1`, shopID).Scan(&connectedEtsyShopID)
	if err != nil {
		writeError(w, http.StatusForbidden, "Najpierw połącz swój sklep z Etsy, aby zweryfikować, że jest Twój")
		return
	}

	var fetched []etsyListing
	result := etsyImportResult{}

	switch req.Mode {
	case "all":
		listings, err := fetchEtsyActiveListings(ctx, connectedEtsyShopID)
		if err != nil {
			writeError(w, http.StatusBadGateway, "Nie udało się pobrać ofert z Etsy: "+err.Error())
			return
		}
		fetched = listings

	case "selected":
		ids := make([]string, 0, len(req.ListingIDs))
		seen := map[string]bool{}
		for _, raw := range req.ListingIDs {
			id := strings.TrimSpace(raw)
			if id == "" || seen[id] {
				continue
			}
			if _, err := strconv.ParseInt(id, 10, 64); err != nil {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			writeError(w, http.StatusBadRequest, "Podaj co najmniej jeden poprawny numer oferty Etsy")
			return
		}
		if len(ids) > etsyMaxSelectedItems {
			ids = ids[:etsyMaxSelectedItems]
		}

		for _, id := range ids {
			listing, err := fetchEtsyListingByID(ctx, id)
			if err != nil {
				result.Errors = append(result.Errors, etsyImportError{EtsyListingID: id, Message: err.Error()})
				continue
			}
			// Ownership check: only allow importing listings that belong to the
			// Etsy shop this seller actually connected via OAuth, so nobody can
			// pull another seller's listings into their own catalog.
			if strconv.FormatInt(listing.ShopID, 10) != connectedEtsyShopID {
				result.Errors = append(result.Errors, etsyImportError{EtsyListingID: id, Message: "Ta oferta nie należy do połączonego sklepu Etsy"})
				continue
			}
			fetched = append(fetched, listing)
		}
	}

	result.Fetched += len(fetched)
	importListings(ctx, shopID, fetched, &result)
	writeJSON(w, http.StatusOK, result)
}

// importListings inserts the given Etsy listings into our shop's catalog,
// skipping ones already imported and recording per-item failures without
// aborting the rest of the batch.
func importListings(ctx context.Context, shopID string, listings []etsyListing, result *etsyImportResult) {
	for _, el := range listings {
		etsyID := strconv.FormatInt(el.ListingID, 10)

		var existingID string
		err := dbPool.QueryRow(ctx, `SELECT id FROM listings WHERE shop_id = $1 AND etsy_listing_id = $2`, shopID, etsyID).Scan(&existingID)
		if err == nil {
			result.Skipped++
			continue
		}

		title := strings.TrimSpace(el.Title)
		if title == "" {
			title = "Oferta z Etsy #" + etsyID
		}
		if len(title) > 255 {
			title = title[:255]
		}

		divisor := el.Price.Divisor
		if divisor == 0 {
			divisor = 100
		}
		price := fmt.Sprintf("%.2f", float64(el.Price.Amount)/float64(divisor))
		currency := strings.ToUpper(strings.TrimSpace(el.Price.CurrencyCode))
		if len(currency) != 3 {
			currency = "USD"
		}

		tx, err := dbPool.Begin(ctx)
		if err != nil {
			result.Errors = append(result.Errors, etsyImportError{EtsyListingID: etsyID, Message: "Błąd bazy danych"})
			continue
		}

		var listingID string
		err = tx.QueryRow(ctx, `
			INSERT INTO listings (shop_id, title, description, price, currency, quantity, is_active, etsy_listing_id)
			VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8)
			RETURNING id
		`, shopID, title, el.Description, price, currency, el.Quantity, el.State == "active", etsyID).Scan(&listingID)
		if err != nil {
			tx.Rollback(ctx)
			result.Errors = append(result.Errors, etsyImportError{EtsyListingID: etsyID, Message: "Nie udało się utworzyć oferty"})
			continue
		}

		photos := make([]string, 0, len(el.Images))
		for _, img := range el.Images {
			if img.URLFullxfull != "" {
				photos = append(photos, img.URLFullxfull)
			}
		}
		if err := replaceListingPhotos(ctx, tx, listingID, photos); err != nil {
			tx.Rollback(ctx)
			result.Errors = append(result.Errors, etsyImportError{EtsyListingID: etsyID, Message: "Nie udało się zapisać zdjęć"})
			continue
		}

		if err := tx.Commit(ctx); err != nil {
			result.Errors = append(result.Errors, etsyImportError{EtsyListingID: etsyID, Message: "Nie udało się sfinalizować oferty"})
			continue
		}

		result.Imported++
	}
}
