package main

import "time"

type ShippingAddress struct {
	FullName   string `json:"full_name"`
	Address    string `json:"address"`
	City       string `json:"city"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
	Phone      string `json:"phone"`
}

type OrderItemRequest struct {
	ListingID string `json:"listing_id"`
	Quantity  int    `json:"quantity"`
}

type CreateOrderRequest struct {
	Items        []OrderItemRequest `json:"items"`
	ShippingAddr ShippingAddress    `json:"shipping_addr"`
	Note         string             `json:"note"`
}

type OrderItemView struct {
	ID            string  `json:"id"`
	ListingID     string  `json:"listing_id"`
	Quantity      int     `json:"quantity"`
	UnitPrice     string  `json:"unit_price"`
	TitleSnapshot string  `json:"title_snapshot"`
	PhotoURL      *string `json:"photo_url,omitempty"`
	Reviewed      bool    `json:"reviewed"`
}

type OrderView struct {
	ID            string          `json:"id"`
	ShopID        string          `json:"shop_id"`
	ShopName      string          `json:"shop_name"`
	BuyerUsername string          `json:"buyer_username,omitempty"`
	Status        string          `json:"status"`
	PaymentStatus string          `json:"payment_status,omitempty"`
	TotalAmount   string          `json:"total_amount"`
	Currency      string          `json:"currency"`
	ShippingAddr  ShippingAddress `json:"shipping_addr"`
	Note          string          `json:"note,omitempty"`
	Items         []OrderItemView `json:"items,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type UpdateOrderStatusRequest struct {
	Status string `json:"status"`
}

type BuyerStats struct {
	TotalOrders  int    `json:"total_orders"`
	TotalSpent   string `json:"total_spent"`
	PendingCount int    `json:"pending_count"`
}

var validOrderStatuses = map[string]bool{
	"pending":    true,
	"paid":       true,
	"processing": true,
	"shipped":    true,
	"delivered":  true,
	"cancelled":  true,
	"refunded":   true,
}
