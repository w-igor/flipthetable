package main

import "time"

// CheckoutSessionRequest requests a Stripe Checkout Session covering one or more
// pending orders belonging to the authenticated buyer.
type CheckoutSessionRequest struct {
	OrderIDs []string `json:"order_ids"`
}

type PaymentView struct {
	ID            string     `json:"id"`
	OrderID       string     `json:"order_id"`
	Provider      string     `json:"provider"`
	ProviderTxID  *string    `json:"provider_tx_id,omitempty"`
	Status        string     `json:"status"`
	Amount        string     `json:"amount"`
	Currency      string     `json:"currency"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
	DeclineReason string     `json:"decline_reason,omitempty"`
}
