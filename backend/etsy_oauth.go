package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	etsyOAuthAuthorizeURL = "https://www.etsy.com/oauth/connect"
	etsyOAuthTokenURL     = "https://api.etsy.com/v3/public/oauth/token"
	etsyOAuthScope        = "listings_r"
	etsyOAuthStateTTL     = 10 * time.Minute
)

func etsyOAuthRedirectURI() string {
	if v := strings.TrimSpace(os.Getenv("ETSY_OAUTH_REDIRECT_URI")); v != "" {
		return v
	}
	return "http://localhost:8080/seller/etsy/oauth/callback"
}

func etsyFrontendURL() string {
	if v := strings.TrimSpace(os.Getenv("FRONTEND_URL")); v != "" {
		return v
	}
	return "http://localhost:3000"
}

type etsyPendingAuth struct {
	ShopID       string
	CodeVerifier string
	CreatedAt    time.Time
}

var (
	etsyOAuthMu      sync.Mutex
	etsyOAuthPending = map[string]etsyPendingAuth{}
)

func etsyOAuthStoreState(shopID, codeVerifier string) string {
	buf := make([]byte, 24)
	_, _ = rand.Read(buf)
	state := base64.RawURLEncoding.EncodeToString(buf)

	etsyOAuthMu.Lock()
	defer etsyOAuthMu.Unlock()
	for k, v := range etsyOAuthPending {
		if time.Since(v.CreatedAt) > etsyOAuthStateTTL {
			delete(etsyOAuthPending, k)
		}
	}
	etsyOAuthPending[state] = etsyPendingAuth{ShopID: shopID, CodeVerifier: codeVerifier, CreatedAt: time.Now()}
	return state
}

func etsyOAuthTakeState(state string) (etsyPendingAuth, bool) {
	etsyOAuthMu.Lock()
	defer etsyOAuthMu.Unlock()
	pending, ok := etsyOAuthPending[state]
	if !ok {
		return etsyPendingAuth{}, false
	}
	delete(etsyOAuthPending, state)
	if time.Since(pending.CreatedAt) > etsyOAuthStateTTL {
		return etsyPendingAuth{}, false
	}
	return pending, true
}

func generatePKCEVerifier() (string, error) {
	buf := make([]byte, 40)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// handleEtsyOAuthStart begins the "Connect with Etsy" flow: the seller is
// redirected to Etsy to grant read access to their own shop, so we can later
// verify import requests actually belong to a shop they proved ownership of.
func handleEtsyOAuthStart(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}
	if etsyAPIKey() == "" {
		writeError(w, http.StatusServiceUnavailable, "Klucz API Etsy nie jest jeszcze skonfigurowany")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	shopID, err := getOwnShopID(ctx, userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Najpierw załóż sklep")
		return
	}

	verifier, err := generatePKCEVerifier()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}
	state := etsyOAuthStoreState(shopID, verifier)

	authURL := etsyOAuthAuthorizeURL + "?" + url.Values{
		"response_type":         {"code"},
		"client_id":             {etsyAPIKey()},
		"redirect_uri":          {etsyOAuthRedirectURI()},
		"scope":                 {etsyOAuthScope},
		"state":                 {state},
		"code_challenge":        {pkceChallenge(verifier)},
		"code_challenge_method": {"S256"},
	}.Encode()

	writeJSON(w, http.StatusOK, map[string]string{"url": authURL})
}

type etsyTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

type etsyShopInfo struct {
	ShopID   int64  `json:"shop_id"`
	ShopName string `json:"shop_name"`
}

func exchangeEtsyCode(ctx context.Context, code, verifier string) (etsyTokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {etsyAPIKey()},
		"redirect_uri":  {etsyOAuthRedirectURI()},
		"code":          {code},
		"code_verifier": {verifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, etsyOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return etsyTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return etsyTokenResponse{}, err
	}
	defer resp.Body.Close()

	var tok etsyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return etsyTokenResponse{}, err
	}
	if resp.StatusCode != http.StatusOK {
		msg := tok.ErrorDesc
		if msg == "" {
			msg = tok.Error
		}
		return etsyTokenResponse{}, fmt.Errorf("Etsy odrzuciło wymianę kodu: %s", msg)
	}
	return tok, nil
}

func fetchEtsyOwnShop(ctx context.Context, etsyUserID, accessToken string) (etsyShopInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, etsyAPIBase+"/users/"+etsyUserID+"/shops", nil)
	if err != nil {
		return etsyShopInfo{}, err
	}
	req.Header.Set("x-api-key", etsyAuthHeader())
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return etsyShopInfo{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return etsyShopInfo{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return etsyShopInfo{}, fmt.Errorf("Etsy API zwróciło błąd (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var shop etsyShopInfo
	if err := json.Unmarshal(body, &shop); err == nil && shop.ShopID != 0 {
		return shop, nil
	}

	var list struct {
		Results []etsyShopInfo `json:"results"`
	}
	if err := json.Unmarshal(body, &list); err == nil && len(list.Results) > 0 {
		return list.Results[0], nil
	}

	return etsyShopInfo{}, fmt.Errorf("Nie znaleziono sklepu Etsy powiązanego z tym kontem")
}

// handleEtsyOAuthCallback is the browser redirect target Etsy sends the user
// back to after they approve (or deny) access. It has no auth header — the
// pending state map is what ties the callback back to our logged-in seller.
func handleEtsyOAuthCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	frontendURL := etsyFrontendURL() + "/pages/dashboard.html"

	if errParam := q.Get("error"); errParam != "" {
		http.Redirect(w, r, frontendURL+"?etsy_error="+url.QueryEscape(errParam), http.StatusFound)
		return
	}

	code := q.Get("code")
	state := q.Get("state")
	if code == "" || state == "" {
		http.Redirect(w, r, frontendURL+"?etsy_error="+url.QueryEscape("Nieprawidłowa odpowiedź z Etsy"), http.StatusFound)
		return
	}

	pending, ok := etsyOAuthTakeState(state)
	if !ok {
		http.Redirect(w, r, frontendURL+"?etsy_error="+url.QueryEscape("Sesja połączenia z Etsy wygasła, spróbuj ponownie"), http.StatusFound)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	tok, err := exchangeEtsyCode(ctx, code, pending.CodeVerifier)
	if err != nil {
		http.Redirect(w, r, frontendURL+"?etsy_error="+url.QueryEscape(err.Error()), http.StatusFound)
		return
	}

	etsyUserID := tok.AccessToken
	if idx := strings.Index(etsyUserID, "."); idx > 0 {
		etsyUserID = etsyUserID[:idx]
	}

	shop, err := fetchEtsyOwnShop(ctx, etsyUserID, tok.AccessToken)
	if err != nil {
		http.Redirect(w, r, frontendURL+"?etsy_error="+url.QueryEscape(err.Error()), http.StatusFound)
		return
	}

	etsyShopID := fmt.Sprintf("%d", shop.ShopID)
	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)

	_, err = dbPool.Exec(ctx, `
		INSERT INTO etsy_connections (shop_id, etsy_shop_id, etsy_user_id, access_token, refresh_token, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (shop_id) DO UPDATE SET
			etsy_shop_id = EXCLUDED.etsy_shop_id,
			etsy_user_id = EXCLUDED.etsy_user_id,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			expires_at = EXCLUDED.expires_at,
			connected_at = NOW()
	`, pending.ShopID, etsyShopID, etsyUserID, tok.AccessToken, tok.RefreshToken, expiresAt)
	if err != nil {
		http.Redirect(w, r, frontendURL+"?etsy_error="+url.QueryEscape("Nie udało się zapisać połączenia z Etsy"), http.StatusFound)
		return
	}
	_, _ = dbPool.Exec(ctx, `UPDATE shops SET etsy_shop_id = $1 WHERE id = $2`, etsyShopID, pending.ShopID)

	http.Redirect(w, r, frontendURL+"?etsy_connected=1", http.StatusFound)
}
