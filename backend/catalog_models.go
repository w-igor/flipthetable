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
	ID        string  `json:"id"`
	URL       string  `json:"url"`
	AltText   *string `json:"alt_text,omitempty"`
	IsPrimary bool    `json:"is_primary"`
	SortOrder int     `json:"sort_order"`
}

type Listing struct {
	ID                string           `json:"id"`
	ShopID            string           `json:"shop_id"`
	ShopName          string           `json:"shop_name"`
	CategoryID        *string          `json:"category_id,omitempty"`
	Title             string           `json:"title"`
	Description       *string          `json:"description,omitempty"`
	Price             string           `json:"price"`
	Currency          string           `json:"currency"`
	Quantity          int              `json:"quantity"`
	IsActive          bool             `json:"is_active"`
	HasVariants       bool             `json:"has_variants"`
	ShippingProfileID *string          `json:"shipping_profile_id,omitempty"`
	Shipping          *ShippingProfile `json:"shipping,omitempty"`
	ViewsCount        int              `json:"views_count"`
	SalesCount        int              `json:"sales_count"`
	AvgRating         *string          `json:"avg_rating,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	PrimaryPhoto      *string          `json:"primary_photo,omitempty"`
	Photos            []ListingPhoto   `json:"photos,omitempty"`
	VariantTypes      []VariantType    `json:"variant_types,omitempty"`
	VariantSkus       []VariantSku     `json:"variant_skus,omitempty"`
}

type ListingsResponse struct {
	Items      []Listing `json:"items"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

type Shop struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description,omitempty"`
	BannerURL   *string   `json:"banner_url,omitempty"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	IsActive    bool      `json:"is_active"`
	SalesCount  int       `json:"sales_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type ShopProfile struct {
	Shop
	ListingsCount int `json:"listings_count"`
}

type ShopRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	BannerURL   string `json:"banner_url"`
	AvatarURL   string `json:"avatar_url"`
}

type SellerStats struct {
	TotalOrders    int            `json:"total_orders"`
	TotalRevenue   string         `json:"total_revenue"`
	ListingsActive int            `json:"listings_active"`
	ListingsTotal  int            `json:"listings_total"`
	OrdersByStatus map[string]int `json:"orders_by_status"`
}

type ListingRequest struct {
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	CategoryID        *string  `json:"category_id"`
	Price             string   `json:"price"`
	Currency          string   `json:"currency"`
	Quantity          int      `json:"quantity"`
	IsActive          *bool    `json:"is_active"`
	Photos            []string `json:"photos"`
	ShippingProfileID *string  `json:"shipping_profile_id"`
}
