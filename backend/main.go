package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

var (
	db        *sql.DB
	jwtSecret = []byte("your-secret-key-change-this-in-prod")
)

func init() {
	var err error
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		os.Getenv("PGUSER"),
		os.Getenv("PGPASSWORD"),
		os.Getenv("PGHOST"),
		os.Getenv("PGPORT"),
		os.Getenv("PGDATABASE"),
	)

	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Database ping failed:", err)
	}

	log.Println("✓ Connected to Neon database")

	// Initialize WebSocket hub
	initWebSocketHub()
	log.Println("✓ WebSocket hub initialized")
}

func main() {
	defer db.Close()

	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("POST /auth/register", corsMiddleware(handleRegister))
	mux.HandleFunc("POST /auth/login", corsMiddleware(handleLogin))
	mux.HandleFunc("POST /auth/refresh", corsMiddleware(handleRefresh))
	mux.HandleFunc("GET /auth/me", corsMiddleware(authMiddleware(handleGetMe)))

	// Listings (Products)
	mux.HandleFunc("GET /api/listings", corsMiddleware(handleGetListings))
	mux.HandleFunc("GET /api/listings/{id}", corsMiddleware(handleGetListing))
	mux.HandleFunc("POST /api/listings", corsMiddleware(authMiddleware(handleCreateListing)))
	mux.HandleFunc("PUT /api/listings/{id}", corsMiddleware(authMiddleware(handleUpdateListing)))
	mux.HandleFunc("DELETE /api/listings/{id}", corsMiddleware(authMiddleware(handleDeleteListing)))

	// Categories
	mux.HandleFunc("GET /api/categories", corsMiddleware(handleGetCategories))

	// Shops
	mux.HandleFunc("GET /api/shops/{id}", corsMiddleware(handleGetShop))
	mux.HandleFunc("POST /api/shops", corsMiddleware(authMiddleware(handleCreateShop)))

	// Orders
	mux.HandleFunc("POST /api/orders", corsMiddleware(authMiddleware(handleCreateOrder)))
	mux.HandleFunc("GET /api/orders", corsMiddleware(authMiddleware(handleGetOrders)))
	mux.HandleFunc("GET /api/orders/{id}", corsMiddleware(authMiddleware(handleGetOrder)))

	// WebSocket
	mux.HandleFunc("GET /ws", authMiddlewareWS(handleWebSocket))

	log.Println("🚀 Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
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
