package main

import "time"

type ShippingProfile struct {
	ID        string    `json:"id"`
	ShopID    string    `json:"shop_id"`
	Name      string    `json:"name"`
	Price     string    `json:"price"`
	MinDays   int       `json:"min_days"`
	MaxDays   int       `json:"max_days"`
	CreatedAt time.Time `json:"created_at"`
}

type ShippingProfileRequest struct {
	Name    string `json:"name"`
	Price   string `json:"price"`
	MinDays int    `json:"min_days"`
	MaxDays int    `json:"max_days"`
}
