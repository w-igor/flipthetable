package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load("../.env"); err != nil {
		if err2 := godotenv.Load(".env"); err2 != nil {
			log.Println("Brak pliku .env — używam zmiennych środowiskowych systemu")
		}
	}

	pool, err := connectDB()
	if err != nil {
		log.Fatalf("Nie udało się połączyć z bazą danych: %v", err)
	}
	defer pool.Close()
	dbPool = pool
	log.Println("Połączono z bazą danych (Neon)")

	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", handleRegister)
	mux.HandleFunc("POST /auth/login", handleLogin)
	mux.HandleFunc("POST /auth/refresh", handleRefresh)
	mux.HandleFunc("GET /auth/me", requireAuth(handleMe))

	mux.HandleFunc("GET /categories", handleGetCategories)
	mux.HandleFunc("GET /listings", handleGetListings)
	mux.HandleFunc("GET /listings/{id}", handleGetListing)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server: http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, withCORS(mux)); err != nil {
		log.Fatalf("Serwer padł: %v", err)
	}
}
