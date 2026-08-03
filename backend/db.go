package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// dbPool is the global PostgreSQL connection pool used by all database operations.
var dbPool *pgxpool.Pool

// connectDB establishes a connection pool to the PostgreSQL database.
// It uses DATABASE_URL if set, or builds the DSN from PGUSER, PGPASSWORD, PGHOST, PGPORT, PGDATABASE.
// Returns an error if the connection cannot be established or if the ping fails.
func connectDB() (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Build DSN from individual environment variables (for Neon or similar)
		dsn = fmt.Sprintf(
			"postgresql://%s:%s@%s:%s/%s?sslmode=require",
			os.Getenv("PGUSER"),
			os.Getenv("PGPASSWORD"),
			os.Getenv("PGHOST"),
			os.Getenv("PGPORT"),
			os.Getenv("PGDATABASE"),
		)
	}

	// Create connection pool with 10-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	// Verify connection is working before returning
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
