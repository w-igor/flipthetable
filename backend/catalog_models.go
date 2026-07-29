package main

import "time"

type Category struct {
	ID          string  `json:"id"`
	ParentID    *string `json:"parent_id,omitempty"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description,omitempty"`
	SortOrder   int     `json:"sort_order"`
}

type ListingPhoto struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	AltText   *string `json:"alt_text,omitempty"`
	IsPrimary bool   `json:"is_primary"`
	SortOrder int    `json:"sort_order"`
}

type Listing struct {
	ID          string         `json:"id"`
	ShopID      string         `json:"shop_id"`
	ShopName    string         `json:"shop_name"`
	CategoryID  *string        `json:"category_id,omitempty"`
	Title       string         `json:"title"`
	Description *string        `json:"description,omitempty"`
	Price       string         `json:"price"`
	Currency    string         `json:"currency"`
	Quantity    int            `json:"quantity"`
	IsActive    bool           `json:"is_active"`
	ViewsCount  int            `json:"views_count"`
	SalesCount  int            `json:"sales_count"`
	AvgRating   *string        `json:"avg_rating,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	PrimaryPhoto *string       `json:"primary_photo,omitempty"`
	Photos      []ListingPhoto `json:"photos,omitempty"`
}

type ListingsResponse struct {
	Items      []Listing `json:"items"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}
