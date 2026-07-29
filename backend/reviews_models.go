package main

import "time"

type Review struct {
	ID               string    `json:"id"`
	ListingID        string    `json:"listing_id"`
	ReviewerID       string    `json:"reviewer_id"`
	ReviewerUsername string    `json:"reviewer_username"`
	OrderItemID      string    `json:"order_item_id"`
	Rating           int       `json:"rating"`
	Comment          *string   `json:"comment,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

type CreateReviewRequest struct {
	OrderItemID string `json:"order_item_id"`
	Rating      int    `json:"rating"`
	Comment     string `json:"comment"`
}
