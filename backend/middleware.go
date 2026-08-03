package main

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// contextKey is a type for context values to avoid collisions with other packages.
type contextKey string

const userIDContextKey contextKey = "userID"

// userIDFromContext retrieves the authenticated user's ID from the request context.
func userIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}

// requireAuth is a middleware that validates JWT access token from Authorization header.
// If valid, it adds the user ID to the context; otherwise returns 401 Unauthorized.
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "Brak tokenu autoryzacji")
			return
		}

		// Extract token from "Bearer <token>" format
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := parseToken(tokenString, "access")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Nieprawidłowy lub wygasły token")
			return
		}

		// Add user ID to request context
		ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
		next(w, r.WithContext(ctx))
	}
}

// requireAdmin is a middleware that extends requireAuth to also check for admin privileges.
// Returns 403 Forbidden if user is not an admin or their account is inactive.
func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// Verify user is admin and account is active
		var isAdmin, isActive bool
		err := dbPool.QueryRow(ctx, `SELECT is_admin, is_active FROM users WHERE id = $1`, userID).Scan(&isAdmin, &isActive)
		if err != nil || !isAdmin || !isActive {
			writeError(w, http.StatusForbidden, "Brak uprawnień administratora")
			return
		}

		next(w, r)
	})
}

// withCORS adds CORS headers to responses allowing cross-origin requests from any origin.
// Handles preflight OPTIONS requests with 204 No Content response.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
