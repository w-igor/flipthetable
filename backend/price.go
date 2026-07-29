package main

import (
	"fmt"
	"strconv"
)

func parsePriceOrZero(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func formatPrice(v float64) string {
	return fmt.Sprintf("%.2f", v)
}
