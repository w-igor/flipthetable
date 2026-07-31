package main

import "time"

type AdminStats struct {
	TotalUsers     int            `json:"total_users"`
	ActiveUsers    int            `json:"active_users"`
	TotalSellers   int            `json:"total_sellers"`
	TotalShops     int            `json:"total_shops"`
	ActiveShops    int            `json:"active_shops"`
	TotalListings  int            `json:"total_listings"`
	ActiveListings int            `json:"active_listings"`
	TotalOrders    int            `json:"total_orders"`
	TotalRevenue   string         `json:"total_revenue"`
	OrdersByStatus map[string]int `json:"orders_by_status"`
}

type AdminUser struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	FullName  *string   `json:"full_name,omitempty"`
	IsSeller  bool      `json:"is_seller"`
	IsAdmin   bool      `json:"is_admin"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminUsersResponse struct {
	Items      []AdminUser `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

type AdminShop struct {
	ID            string    `json:"id"`
	OwnerID       string    `json:"owner_id"`
	OwnerUsername string    `json:"owner_username"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	IsActive      bool      `json:"is_active"`
	SalesCount    int       `json:"sales_count"`
	ListingsCount int       `json:"listings_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type AdminShopsResponse struct {
	Items      []AdminShop `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

type AdminListing struct {
	ID             string    `json:"id"`
	ShopID         string    `json:"shop_id"`
	ShopName       string    `json:"shop_name"`
	SellerUsername string    `json:"seller_username"`
	Title          string    `json:"title"`
	Price          string    `json:"price"`
	Currency       string    `json:"currency"`
	Quantity       int       `json:"quantity"`
	IsActive       bool      `json:"is_active"`
	SalesCount     int       `json:"sales_count"`
	CreatedAt      time.Time `json:"created_at"`
}

type AdminListingsResponse struct {
	Items      []AdminListing `json:"items"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

type AdminOrdersResponse struct {
	Items      []OrderView `json:"items"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

type SetActiveRequest struct {
	IsActive bool `json:"is_active"`
}

type SetAdminRequest struct {
	IsAdmin bool `json:"is_admin"`
}

type AdminAuditLogEntry struct {
	ID            string    `json:"id"`
	AdminID       string    `json:"admin_id"`
	AdminUsername string    `json:"admin_username"`
	Action        string    `json:"action"`
	TargetType    string    `json:"target_type"`
	TargetID      *string   `json:"target_id,omitempty"`
	Details       *string   `json:"details,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type AdminAuditLogResponse struct {
	Items      []AdminAuditLogEntry `json:"items"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
	TotalPages int                  `json:"total_pages"`
}

type CategoryRequest struct {
	Name        string  `json:"name"`
	ParentID    *string `json:"parent_id"`
	Description string  `json:"description"`
	SortOrder   int     `json:"sort_order"`
}
