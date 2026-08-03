package main

import "context"

// runStartupMigrations applies small additive schema changes that shipped after the
// original init.sql, so an already-provisioned database catches up automatically on boot.
func runStartupMigrations() error {
	ctx := context.Background()
	statements := []string{
		`ALTER TABLE shops ADD COLUMN IF NOT EXISTS etsy_shop_id VARCHAR(100)`,
		`ALTER TABLE listings ADD COLUMN IF NOT EXISTS etsy_listing_id VARCHAR(100)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_listings_shop_etsy_id ON listings (shop_id, etsy_listing_id) WHERE etsy_listing_id IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS etsy_connections (
			shop_id       UUID        PRIMARY KEY REFERENCES shops(id) ON DELETE CASCADE,
			etsy_shop_id  VARCHAR(100) NOT NULL,
			etsy_user_id  VARCHAR(100) NOT NULL,
			access_token  TEXT        NOT NULL,
			refresh_token TEXT        NOT NULL,
			expires_at    TIMESTAMPTZ NOT NULL,
			connected_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}
	for _, stmt := range statements {
		if _, err := dbPool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
