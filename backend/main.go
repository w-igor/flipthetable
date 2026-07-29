package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var jwtSecret = []byte("your-secret-key-change-this-in-prod")

type User struct {
	ID       int       `json:"id"`
	Email    string    `json:"email"`
	Password string    `json:"-"`
	CreateAt time.Time `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func init() {
	var err error
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:5432/flipthetable?sslmode=disable",
		os.Getenv("PGUSER"),
		os.Getenv("PGPASSWORD"),
	)

	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	log.Println("✓ Connected to database")
}

func main() {
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /auth/register", corsMiddleware(handleRegister))
	mux.HandleFunc("POST /auth/login", corsMiddleware(handleLogin))
	mux.HandleFunc("POST /auth/refresh", corsMiddleware(handleRefresh))
	mux.HandleFunc("GET /auth/me", corsMiddleware(authMiddleware(handleMe)))

	log.Println("🚀 Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password required", http.StatusBadRequest)
		return
	}

	hashedPwd, _ := bcrypt.GenerateFromPassword([]byte(req.Password), 10)

	var id int
	err := db.QueryRow(
		"INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id",
		req.Email, string(hashedPwd),
	).Scan(&id)

	if err != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	accessToken, _ := generateAccessToken(req.Email)
	refreshToken, _ := generateRefreshToken(req.Email)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: User{
			ID:    id,
			Email: req.Email,
		},
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var user User
	err := db.QueryRow("SELECT id, email, password FROM users WHERE email = $1", req.Email).
		Scan(&user.ID, &user.Email, &user.Password)

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	accessToken, _ := generateAccessToken(req.Email)
	refreshToken, _ := generateRefreshToken(req.Email)

	if req.Remember {
		db.Exec(
			"INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
			user.ID, refreshToken, time.Now().AddDate(0, 0, 30),
		)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}

func handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	json.NewDecoder(r.Body).Decode(&req)

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(req["refresh_token"], claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	accessToken, _ := generateAccessToken(claims.Email)
	refreshToken, _ := generateRefreshToken(claims.Email)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")
	var user User
	db.QueryRow("SELECT id, email, created_at FROM users WHERE email = $1", email).
		Scan(&user.ID, &user.Email, &user.CreateAt)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func generateAccessToken(email string) (string, error) {
	claims := &Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func generateRefreshToken(email string) (string, error) {
	claims := &Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().AddDate(0, 0, 30)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		token := parts[1]
		claims := &Claims{}
		_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add email to request context for next handler
		ctx := context.WithValue(r.Context(), "email", claims.Email)
		r.Header.Set("X-Email", claims.Email)
		next(w, r.WithContext(ctx))
	}
}
