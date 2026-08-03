package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// declinedTestCard is a test card number that simulates a payment failure,
// allowing easy testing of the payment decline flow without a real payment provider.
const declinedTestCard = "4000000000000002"

// insertPendingPayment creates a new payment record with "pending" status in a transaction.
func insertPendingPayment(ctx context.Context, tx pgx.Tx, orderID string, amount float64, currency string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO payments (order_id, provider, status, amount, currency)
		VALUES ($1, 'card', 'pending', $2, $3)
	`, orderID, amount, currency)
	return err
}

// generateTxID creates a random transaction ID for payment records.
func generateTxID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "tx_" + hex.EncodeToString(b)
}

// luhnValid performs Luhn algorithm validation on a credit card number.
// This is a basic check to catch typos before sending to a payment provider.
func luhnValid(number string) bool {
	sum := 0
	alt := false
	for i := len(number) - 1; i >= 0; i-- {
		c := number[i]
		if c < '0' || c > '9' {
			return false
		}
		d := int(c - '0')
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}
	return sum%10 == 0
}

// handlePayOrder processes a payment attempt for a pending order.
// Validates card details, checks order ownership and status, then marks payment as completed or failed.
// The test card (4000000000000002) always fails; all others succeed.
func handlePayOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}
	orderID := r.PathValue("id")

	var req PayOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	// Validate card number format and Luhn checksum
	cardNumber := strings.ReplaceAll(req.CardNumber, " ", "")
	if len(cardNumber) < 13 || len(cardNumber) > 19 || !luhnValid(cardNumber) {
		writeError(w, http.StatusBadRequest, "Nieprawidłowy numer karty")
		return
	}
	// Validate CVC (usually 3 or 4 digits)
	if len(req.CVC) < 3 || len(req.CVC) > 4 {
		writeError(w, http.StatusBadRequest, "Nieprawidłowy kod CVC")
		return
	}
	// Check if card expiration date is in the future
	now := time.Now()
	if req.ExpYear < now.Year() || (req.ExpYear == now.Year() && req.ExpMonth < int(now.Month())) {
		writeError(w, http.StatusBadRequest, "Karta wygasła")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Start a database transaction to ensure atomicity of payment + order status updates
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd bazy danych")
		return
	}
	defer tx.Rollback(ctx)

	// Fetch order and payment details with row-level lock to prevent concurrent payment attempts
	var buyerID, orderStatus string
	var paymentID, paymentStatus, amount string
	err = tx.QueryRow(ctx, `
		SELECT o.buyer_id, o.status, p.id, p.status, o.total_amount
		FROM orders o
		JOIN payments p ON p.order_id = o.id
		WHERE o.id = $1
		FOR UPDATE OF o
	`, orderID).Scan(&buyerID, &orderStatus, &paymentID, &paymentStatus, &amount)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "Zamówienie nie znalezione")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}

	// Verify the user is the order buyer
	if buyerID != userID {
		writeError(w, http.StatusForbidden, "To nie jest Twoje zamówienie")
		return
	}
	// Prevent double payments
	if paymentStatus == "completed" {
		writeError(w, http.StatusConflict, "Zamówienie jest już opłacone")
		return
	}
	// Only allow payment on pending orders
	if orderStatus != "pending" {
		writeError(w, http.StatusConflict, "Tego zamówienia nie można już opłacić")
		return
	}

	// Simulate payment decline for the test card
	if cardNumber == declinedTestCard {
		if _, err := tx.Exec(ctx, `UPDATE payments SET status = 'failed' WHERE id = $1`, paymentID); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd zapisu płatności")
			return
		}
		if err := tx.Commit(ctx); err != nil {
			writeError(w, http.StatusInternalServerError, "Nie udało się zapisać płatności")
			return
		}
		writeJSON(w, http.StatusPaymentRequired, map[string]string{
			"message":        "Płatność odrzucona przez bank",
			"payment_status": "failed",
		})
		return
	}

	// Mark payment as completed with transaction ID and timestamp
	txID := generateTxID()
	if _, err := tx.Exec(ctx, `
		UPDATE payments SET status = 'completed', provider_tx_id = $1, paid_at = NOW() WHERE id = $2
	`, txID, paymentID); err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd zapisu płatności")
		return
	}
	// Update order status to paid
	if _, err := tx.Exec(ctx, `UPDATE orders SET status = 'paid' WHERE id = $1`, orderID); err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd aktualizacji zamówienia")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać płatności")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message":        "Płatność zaakceptowana",
		"payment_status": "completed",
		"order_status":   "paid",
	})
}
