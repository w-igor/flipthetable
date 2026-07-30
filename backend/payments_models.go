package main

import "time"

type PayOrderRequest struct {
	CardholderName string `json:"cardholder_name"`
	CardNumber     string `json:"card_number"`
	ExpMonth       int    `json:"exp_month"`
	ExpYear        int    `json:"exp_year"`
	CVC            string `json:"cvc"`
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
