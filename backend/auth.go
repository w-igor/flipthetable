package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Message: message})
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if len(req.Username) < 3 {
		writeError(w, http.StatusBadRequest, "Nazwa użytkownika musi mieć min. 3 znaki")
		return
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "Podaj prawidłowy adres e-mail")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "Hasło musi mieć min. 8 znaków")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się przetworzyć hasła")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user User
	err = dbPool.QueryRow(ctx, `
		INSERT INTO users (email, username, password_hash, is_seller)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, username, full_name, avatar_url, is_seller, is_admin, is_active, created_at
	`, req.Email, req.Username, string(passwordHash), req.IsSeller).Scan(
		&user.ID, &user.Email, &user.Username, &user.FullName, &user.AvatarURL,
		&user.IsSeller, &user.IsAdmin, &user.IsActive, &user.CreatedAt,
	)

	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "Użytkownik z tym adresem e-mail lub nazwą już istnieje")
			return
		}
		writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć konta")
		return
	}

	accessToken, refreshToken, err := generateAuthTokens(user.ID, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się wygenerować tokenów")
		return
	}

	writeJSON(w, http.StatusCreated, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user User
	var passwordHash string
	err := dbPool.QueryRow(ctx, `
		SELECT id, email, username, full_name, avatar_url, password_hash, is_seller, is_admin, is_active, created_at
		FROM users WHERE email = $1
	`, req.Email).Scan(
		&user.ID, &user.Email, &user.Username, &user.FullName, &user.AvatarURL,
		&passwordHash, &user.IsSeller, &user.IsAdmin, &user.IsActive, &user.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		writeError(w, http.StatusUnauthorized, "Nieprawidłowy e-mail lub hasło")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}

	if !user.IsActive {
		writeError(w, http.StatusForbidden, "Konto zostało dezaktywowane")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "Nieprawidłowy e-mail lub hasło")
		return
	}

	accessToken, refreshToken, err := generateAuthTokens(user.ID, req.Remember)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się wygenerować tokenów")
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	claims, err := parseToken(req.RefreshToken, "refresh")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Nieprawidłowy lub wygasły refresh token")
		return
	}

	accessToken, err := generateToken(claims.UserID, "access", accessTokenTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się wygenerować tokenu")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"access_token": accessToken})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var user User
	err := dbPool.QueryRow(ctx, `
		SELECT id, email, username, full_name, avatar_url, is_seller, is_admin, is_active, created_at
		FROM users WHERE id = $1
	`, userID).Scan(
		&user.ID, &user.Email, &user.Username, &user.FullName, &user.AvatarURL,
		&user.IsSeller, &user.IsAdmin, &user.IsActive, &user.CreatedAt,
	)

	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "Użytkownik nie istnieje")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func isUniqueViolation(err error) bool {
	return strings.Contains(err.Error(), "duplicate key value violates unique constraint")
}
