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
	mux.HandleFunc("POST /listings", requireAuth(handleCreateListing))
	mux.HandleFunc("PUT /listings/{id}", requireAuth(handleUpdateListing))
	mux.HandleFunc("DELETE /listings/{id}", requireAuth(handleDeleteListing))
	mux.HandleFunc("GET /listings/{id}/reviews", handleGetListingReviews)
	mux.HandleFunc("POST /reviews", requireAuth(handleCreateReview))

	mux.HandleFunc("POST /shops", requireAuth(handleCreateShop))
	mux.HandleFunc("GET /shops/me", requireAuth(handleGetMyShop))
	mux.HandleFunc("PUT /shops/me", requireAuth(handleUpdateShop))
	mux.HandleFunc("GET /shops/{id}", handleGetShopPublic)

	mux.HandleFunc("GET /seller/listings", requireAuth(handleGetMyListings))
	mux.HandleFunc("GET /seller/stats", requireAuth(handleGetSellerStats))
	mux.HandleFunc("GET /seller/orders", requireAuth(handleListSellerOrders))
	mux.HandleFunc("PUT /seller/orders/{id}/status", requireAuth(handleUpdateOrderStatus))

	mux.HandleFunc("POST /messages", requireAuth(handleSendMessage))
	mux.HandleFunc("GET /messages/conversations", requireAuth(handleListConversations))
	mux.HandleFunc("GET /messages/unread-count", requireAuth(handleUnreadMessageCount))
	mux.HandleFunc("GET /messages/with/{userId}", requireAuth(handleGetThread))

	mux.HandleFunc("GET /favorites", requireAuth(handleListFavorites))
	mux.HandleFunc("GET /favorites/ids", requireAuth(handleListFavoriteIDs))
	mux.HandleFunc("POST /favorites", requireAuth(handleAddFavorite))
	mux.HandleFunc("DELETE /favorites/{listingId}", requireAuth(handleRemoveFavorite))

	mux.HandleFunc("POST /uploads", requireAuth(handleUploadPhoto))
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))))

	mux.HandleFunc("POST /orders", requireAuth(handleCreateOrder))
	mux.HandleFunc("GET /orders", requireAuth(handleListOrders))
	mux.HandleFunc("GET /orders/stats", requireAuth(handleGetBuyerStats))
	mux.HandleFunc("GET /orders/{id}", requireAuth(handleGetOrder))
	mux.HandleFunc("POST /orders/{id}/pay", requireAuth(handlePayOrder))

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
