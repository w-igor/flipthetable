package main

import (
	"fmt"
	"strconv"
)

// parsePriceOrZero attempts to parse a string as a float64 price, returning 0 on parse failure.
// Used for gracefully handling invalid price inputs without throwing errors.
func parsePriceOrZero(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// formatPrice formats a float64 price to exactly 2 decimal places (e.g., "19.99").
func formatPrice(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
