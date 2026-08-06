package main

import "context"

// runStartupMigrations applies schema changes (migrations) that were added after
// the initial database setup. These are idempotent (safe to run repeatedly) and
// allow already-provisioned databases to auto-update on server boot.
// Currently includes Etsy OAuth integration schema (columns and tables).
func runStartupMigrations() error {
	ctx := context.Background()
	statements := []string{
		// Add Etsy shop ID column to track which Etsy shop is synced to our shop
		`ALTER TABLE shops ADD COLUMN IF NOT EXISTS etsy_shop_id VARCHAR(100)`,
		// Add Etsy listing ID column to link our listings to Etsy listings for syncing
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS etsy_listing_id VARCHAR(100)`,
		// Index to prevent duplicate Etsy listings in the same shop
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_listings_shop_etsy_id ON listings (shop_id, etsy_listing_id) WHERE etsy_listing_id IS NOT NULL`,
		// Table to store Etsy OAuth credentials and connection details
		`CREATE TABLE IF NOT EXISTS etsy_connections (
			shop_id       UUID        PRIMARY KEY REFERENCES shops(id) ON DELETE CASCADE,
			etsy_shop_id  VARCHAR(100) NOT NULL,
			etsy_user_id  VARCHAR(100) NOT NULL,
			access_token  TEXT        NOT NULL,
			refresh_token TEXT        NOT NULL,
			expires_at    TIMESTAMPTZ NOT NULL,
			connected_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		// Stripe Checkout integration: track which session a payment belongs to,
		// so the webhook handler can look up and update the right orders/payments.
		`ALTER TABLE payments ADD COLUMN IF NOT EXISTS stripe_checkout_session_id VARCHAR(255)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_stripe_session_id ON payments (stripe_checkout_session_id)`,
	}
	for _, stmt := range statements {
		if _, err := dbPool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
