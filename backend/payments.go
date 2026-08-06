package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
)

// insertPendingPayment creates a new payment record with "pending" status in a transaction.
func insertPendingPayment(ctx context.Context, tx pgx.Tx, orderID string, amount float64, currency string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO payments (order_id, provider, status, amount, currency)
		VALUES ($1, 'stripe', 'pending', $2, $3)
	`, orderID, amount, currency)
	return err
}

// checkoutOrderRow holds the order/payment state needed to validate and build a Stripe Checkout Session.
type checkoutOrderRow struct {
	BuyerID       string
	Status        string
	ShopName      string
	PaymentID     string
	PaymentStatus string
	Currency      string
}

// handleCreateCheckoutSession builds a Stripe Checkout Session covering one or more pending
// orders belonging to the authenticated buyer (a single checkout can span several shops/orders).
// The buyer is redirected to the Stripe-hosted payment page; order status is only updated once
// Stripe confirms payment via the /webhooks/stripe endpoint.
func handleCreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	var req CheckoutSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.OrderIDs) == 0 {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := dbPool.Query(ctx, `
		SELECT o.id, o.buyer_id, o.status, s.name, p.id, p.status, o.currency
		FROM orders o
		JOIN shops s ON s.id = o.shop_id
		JOIN payments p ON p.order_id = o.id
		WHERE o.id = ANY($1::uuid[])
	`, req.OrderIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd bazy danych")
		return
	}
	orders := map[string]checkoutOrderRow{}
	for rows.Next() {
		var id string
		var row checkoutOrderRow
		if err := rows.Scan(&id, &row.BuyerID, &row.Status, &row.ShopName, &row.PaymentID, &row.PaymentStatus, &row.Currency); err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "Błąd odczytu zamówień")
			return
		}
		orders[id] = row
	}
	rows.Close()

	if len(orders) != len(req.OrderIDs) {
		writeError(w, http.StatusNotFound, "Zamówienie nie znalezione")
		return
	}

	lineItems := []*stripe.CheckoutSessionLineItemParams{}
	currency := "pln"
	for _, orderID := range req.OrderIDs {
		o := orders[orderID]
		if o.BuyerID != userID {
			writeError(w, http.StatusForbidden, "To nie jest Twoje zamówienie")
			return
		}
		if o.PaymentStatus == "completed" {
			writeError(w, http.StatusConflict, "Zamówienie jest już opłacone")
			return
		}
		if o.Status != "pending" {
			writeError(w, http.StatusConflict, "Tego zamówienia nie można już opłacić")
			return
		}
		if o.Currency != "" {
			currency = strings.ToLower(o.Currency)
		}

		items := fetchOrderItems(ctx, orderID)
		for _, item := range items {
			name := item.TitleSnapshot
			if item.VariantLabel != nil && *item.VariantLabel != "" {
				name += " (" + *item.VariantLabel + ")"
			}
			lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
				Quantity: stripe.Int64(int64(item.Quantity)),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(currency),
					UnitAmount: stripe.Int64(priceToCents(parsePriceOrZero(item.UnitPrice))),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(name + " — " + o.ShopName),
					},
				},
			})
		}

		var shippingAmount string
		if err := dbPool.QueryRow(ctx, `SELECT shipping_amount FROM orders WHERE id = $1`, orderID).Scan(&shippingAmount); err == nil {
			if amount := parsePriceOrZero(shippingAmount); amount > 0 {
				lineItems = append(lineItems, &stripe.CheckoutSessionLineItemParams{
					Quantity: stripe.Int64(1),
					PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
						Currency:   stripe.String(currency),
						UnitAmount: stripe.Int64(priceToCents(amount)),
						ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
							Name: stripe.String("Wysyłka — " + o.ShopName),
						},
					},
				})
			}
		}
	}

	if len(lineItems) == 0 {
		writeError(w, http.StatusBadRequest, "Brak pozycji do opłacenia")
		return
	}

	orderIDsJoined := strings.Join(req.OrderIDs, ",")
	base := etsyFrontendURL()

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems:         lineItems,
		ClientReferenceID: stripe.String(userID),
		SuccessURL:        stripe.String(base + "/pages/order-confirmation.html?ids=" + orderIDsJoined + "&session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:         stripe.String(base + "/pages/orders.html?paymentCancelled=1"),
		Metadata:          map[string]string{"order_ids": orderIDsJoined, "buyer_id": userID},
	}

	sess, err := session.New(params)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Nie udało się utworzyć sesji płatności Stripe")
		return
	}

	if _, err := dbPool.Exec(ctx, `
		UPDATE payments SET stripe_checkout_session_id = $1 WHERE order_id = ANY($2::uuid[])
	`, sess.ID, req.OrderIDs); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zapisać sesji płatności")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": sess.URL})
}

// priceToCents converts a decimal PLN amount to the smallest currency unit (grosz) that Stripe expects.
func priceToCents(amount float64) int64 {
	return int64(amount*100 + 0.5)
}

// markCheckoutSessionOrders updates all orders/payments tied to a Stripe Checkout Session to the given
// terminal status. Only rows still pending are touched, making the handler safe against duplicate webhook deliveries.
func markCheckoutSessionOrders(ctx context.Context, sessionID string, paymentIntentID string, succeeded bool) error {
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if succeeded {
		if _, err := tx.Exec(ctx, `
			UPDATE payments SET status = 'completed', provider_tx_id = $1, paid_at = NOW()
			WHERE stripe_checkout_session_id = $2 AND status = 'pending'
		`, paymentIntentID, sessionID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE orders SET status = 'paid'
			WHERE status = 'pending' AND id IN (SELECT order_id FROM payments WHERE stripe_checkout_session_id = $1)
		`, sessionID); err != nil {
			return err
		}
	} else {
		if _, err := tx.Exec(ctx, `
			UPDATE payments SET status = 'failed'
			WHERE stripe_checkout_session_id = $1 AND status = 'pending'
		`, sessionID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// handleStripeWebhook receives asynchronous payment confirmation events from Stripe.
// It verifies the request signature before trusting any payload, then marks the
// orders/payments tied to the Checkout Session as paid or failed.
func handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	const maxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), stripeWebhookSecret())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch event.Type {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if sess.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
			break
		}
		paymentIntentID := ""
		if sess.PaymentIntent != nil {
			paymentIntentID = sess.PaymentIntent.ID
		}
		if err := markCheckoutSessionOrders(ctx, sess.ID, paymentIntentID, true); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case "checkout.session.async_payment_failed", "checkout.session.expired":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := markCheckoutSessionOrders(ctx, sess.ID, "", false); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
