package main

import (
	"os"
	"strings"
)

// stripeSecretKey returns the Stripe secret API key used to call the Stripe API.
func stripeSecretKey() string {
	return strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
}

// stripeWebhookSecret returns the signing secret used to verify Stripe webhook requests.
func stripeWebhookSecret() string {
	return strings.TrimSpace(os.Getenv("STRIPE_WEBHOOK_SECRET"))
}
