package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// logAdminAction records an audit trail entry when an admin performs an action.
// Uses best-effort error handling: logs are recorded separately and don't block the admin action.
// Useful for compliance and security investigations.
func logAdminAction(ctx context.Context, adminID, action, targetType, targetID, details string) {
	_, err := dbPool.Exec(ctx, `
		INSERT INTO admin_audit_log (admin_id, action, target_type, target_id, details)
		VALUES ($1, $2, $3, NULLIF($4, '')::uuid, NULLIF($5, ''))
	`, adminID, action, targetType, targetID, details)
	if err != nil {
		log.Println("nie udało się zapisać wpisu audytu:", err)
	}
}

// paginationParams extracts page and page_size query parameters with sensible defaults.
// Returns page >= 1 and pageSize capped to 1-100 range.
func paginationParams(q map[string][]string) (page, pageSize int) {
	get := func(key string) string {
		if v, ok := q[key]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	page, _ = strconv.Atoi(get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ = strconv.Atoi(get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return
}

// handleAdminStats returns platform-wide statistics for the admin dashboard:
// user counts, shop counts, listing counts, and order metrics.
func handleAdminStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	stats := AdminStats{OrdersByStatus: map[string]int{}}

	err := dbPool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active), COUNT(*) FILTER (WHERE is_seller)
		FROM users
	`).Scan(&stats.TotalUsers, &stats.ActiveUsers, &stats.TotalSellers)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać statystyk użytkowników")
		return
	}

	err = dbPool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active) FROM shops
	`).Scan(&stats.TotalShops, &stats.ActiveShops)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać statystyk sklepów")
		return
	}

	err = dbPool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE is_active) FROM listings
	`).Scan(&stats.TotalListings, &stats.ActiveListings)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać statystyk ofert")
		return
	}

	var totalRevenue float64
	err = dbPool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(total_amount) FILTER (WHERE status NOT IN ('cancelled', 'refunded')), 0)
		FROM orders
	`).Scan(&stats.TotalOrders, &totalRevenue)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać statystyk zamówień")
		return
	}
	stats.TotalRevenue = formatPrice(totalRevenue)

	statusRows, err := dbPool.Query(ctx, `SELECT status, COUNT(*) FROM orders GROUP BY status`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać statystyk statusów")
		return
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var status string
		var count int
		if err := statusRows.Scan(&status, &count); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu statystyk")
			return
		}
		stats.OrdersByStatus[status] = count
	}

	writeJSON(w, http.StatusOK, stats)
}

func handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, pageSize := paginationParams(q)

	conditions := []string{"1=1"}
	args := []interface{}{}
	argN := 0
	nextArg := func(v interface{}) string {
		argN++
		args = append(args, v)
		return "$" + strconv.Itoa(argN)
	}

	if search := strings.TrimSpace(q.Get("q")); search != "" {
		placeholder := nextArg("%" + search + "%")
		conditions = append(conditions, "(username ILIKE "+placeholder+" OR email ILIKE "+placeholder+")")
	}
	whereClause := strings.Join(conditions, " AND ")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var total int
	if err := dbPool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE "+whereClause, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się policzyć użytkowników")
		return
	}

	limitArg := nextArg(pageSize)
	offsetArg := nextArg((page - 1) * pageSize)

	rows, err := dbPool.Query(ctx, `
		SELECT id, email, username, full_name, is_seller, is_admin, is_active, created_at
		FROM users
		WHERE `+whereClause+`
		ORDER BY created_at DESC
		LIMIT `+limitArg+` OFFSET `+offsetArg, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać użytkowników")
		return
	}
	defer rows.Close()

	items := []AdminUser{}
	for rows.Next() {
		var u AdminUser
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.FullName, &u.IsSeller, &u.IsAdmin, &u.IsActive, &u.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu użytkowników")
			return
		}
		items = append(items, u)
	}

	writeJSON(w, http.StatusOK, AdminUsersResponse{
		Items: items, Total: total, Page: page, PageSize: pageSize,
		TotalPages: totalPagesOf(total, pageSize),
	})
}

func handleAdminSetUserActive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	adminID, _ := userIDFromContext(r.Context())
	if id == adminID {
		writeError(w, http.StatusBadRequest, "Nie możesz zablokować własnego konta")
		return
	}

	var req SetActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tag, err := dbPool.Exec(ctx, `UPDATE users SET is_active = $1 WHERE id = $2`, req.IsActive, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować użytkownika")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Użytkownik nie znaleziony")
		return
	}

	action := "user.deactivate"
	if req.IsActive {
		action = "user.activate"
	}
	logAdminAction(ctx, adminID, action, "user", id, "")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Zaktualizowano użytkownika"})
}

func handleAdminSetUserAdmin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	adminID, _ := userIDFromContext(r.Context())
	if id == adminID {
		writeError(w, http.StatusBadRequest, "Nie możesz zmienić własnych uprawnień administratora")
		return
	}

	var req SetAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tag, err := dbPool.Exec(ctx, `UPDATE users SET is_admin = $1 WHERE id = $2`, req.IsAdmin, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować uprawnień")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Użytkownik nie znaleziony")
		return
	}

	action := "user.revoke_admin"
	if req.IsAdmin {
		action = "user.grant_admin"
	}
	logAdminAction(ctx, adminID, action, "user", id, "")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Zaktualizowano uprawnienia"})
}

func handleAdminListShops(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, pageSize := paginationParams(q)

	conditions := []string{"1=1"}
	args := []interface{}{}
	argN := 0
	nextArg := func(v interface{}) string {
		argN++
		args = append(args, v)
		return "$" + strconv.Itoa(argN)
	}

	if search := strings.TrimSpace(q.Get("q")); search != "" {
		conditions = append(conditions, "s.name ILIKE "+nextArg("%"+search+"%"))
	}
	whereClause := strings.Join(conditions, " AND ")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var total int
	countQuery := `SELECT COUNT(*) FROM shops s WHERE ` + whereClause
	if err := dbPool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się policzyć sklepów")
		return
	}

	limitArg := nextArg(pageSize)
	offsetArg := nextArg((page - 1) * pageSize)

	rows, err := dbPool.Query(ctx, `
		SELECT s.id, s.owner_id, u.username, s.name, s.slug, s.is_active, s.sales_count, s.created_at,
		       (SELECT COUNT(*) FROM listings l WHERE l.shop_id = s.id)
		FROM shops s
		JOIN users u ON u.id = s.owner_id
		WHERE `+whereClause+`
		ORDER BY s.created_at DESC
		LIMIT `+limitArg+` OFFSET `+offsetArg, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać sklepów")
		return
	}
	defer rows.Close()

	items := []AdminShop{}
	for rows.Next() {
		var s AdminShop
		if err := rows.Scan(&s.ID, &s.OwnerID, &s.OwnerUsername, &s.Name, &s.Slug, &s.IsActive, &s.SalesCount, &s.CreatedAt, &s.ListingsCount); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu sklepów")
			return
		}
		items = append(items, s)
	}

	writeJSON(w, http.StatusOK, AdminShopsResponse{
		Items: items, Total: total, Page: page, PageSize: pageSize,
		TotalPages: totalPagesOf(total, pageSize),
	})
}

func handleAdminSetShopActive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	adminID, _ := userIDFromContext(r.Context())

	var req SetActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tag, err := dbPool.Exec(ctx, `UPDATE shops SET is_active = $1 WHERE id = $2`, req.IsActive, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować sklepu")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Sklep nie znaleziony")
		return
	}

	action := "shop.deactivate"
	if req.IsActive {
		action = "shop.activate"
	}
	logAdminAction(ctx, adminID, action, "shop", id, "")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Zaktualizowano sklep"})
}

func handleAdminListListings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, pageSize := paginationParams(q)

	conditions := []string{"1=1"}
	args := []interface{}{}
	argN := 0
	nextArg := func(v interface{}) string {
		argN++
		args = append(args, v)
		return "$" + strconv.Itoa(argN)
	}

	if search := strings.TrimSpace(q.Get("q")); search != "" {
		conditions = append(conditions, "l.title ILIKE "+nextArg("%"+search+"%"))
	}
	if q.Get("active") == "true" {
		conditions = append(conditions, "l.is_active = TRUE")
	} else if q.Get("active") == "false" {
		conditions = append(conditions, "l.is_active = FALSE")
	}
	whereClause := strings.Join(conditions, " AND ")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var total int
	countQuery := `
		SELECT COUNT(*) FROM listings l
		JOIN shops s ON s.id = l.shop_id
		WHERE ` + whereClause
	if err := dbPool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się policzyć ofert")
		return
	}

	limitArg := nextArg(pageSize)
	offsetArg := nextArg((page - 1) * pageSize)

	rows, err := dbPool.Query(ctx, `
		SELECT l.id, l.shop_id, s.name, u.username, l.title, l.price, l.currency,
		       l.quantity, l.is_active, l.sales_count, l.created_at
		FROM listings l
		JOIN shops s ON s.id = l.shop_id
		JOIN users u ON u.id = s.owner_id
		WHERE `+whereClause+`
		ORDER BY l.created_at DESC
		LIMIT `+limitArg+` OFFSET `+offsetArg, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać ofert")
		return
	}
	defer rows.Close()

	items := []AdminListing{}
	for rows.Next() {
		var l AdminListing
		if err := rows.Scan(&l.ID, &l.ShopID, &l.ShopName, &l.SellerUsername, &l.Title, &l.Price, &l.Currency, &l.Quantity, &l.IsActive, &l.SalesCount, &l.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu ofert")
			return
		}
		items = append(items, l)
	}

	writeJSON(w, http.StatusOK, AdminListingsResponse{
		Items: items, Total: total, Page: page, PageSize: pageSize,
		TotalPages: totalPagesOf(total, pageSize),
	})
}

func handleAdminSetListingActive(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	adminID, _ := userIDFromContext(r.Context())

	var req SetActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tag, err := dbPool.Exec(ctx, `UPDATE listings SET is_active = $1 WHERE id = $2`, req.IsActive, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować oferty")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Oferta nie znaleziona")
		return
	}

	action := "listing.deactivate"
	if req.IsActive {
		action = "listing.activate"
	}
	logAdminAction(ctx, adminID, action, "listing", id, "")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Zaktualizowano ofertę"})
}

func handleAdminListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, pageSize := paginationParams(q)

	conditions := []string{"1=1"}
	args := []interface{}{}
	argN := 0
	nextArg := func(v interface{}) string {
		argN++
		args = append(args, v)
		return "$" + strconv.Itoa(argN)
	}

	if status := strings.TrimSpace(q.Get("status")); status != "" && validOrderStatuses[status] {
		conditions = append(conditions, "o.status = "+nextArg(status))
	}
	whereClause := strings.Join(conditions, " AND ")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var total int
	if err := dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM orders o WHERE `+whereClause, args...).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się policzyć zamówień")
		return
	}

	limitArg := nextArg(pageSize)
	offsetArg := nextArg((page - 1) * pageSize)

	rows, err := dbPool.Query(ctx, `
		SELECT o.id, o.shop_id, s.name, u.username, o.status, o.total_amount, o.shipping_amount, o.currency,
		       o.shipping_addr, o.note, o.created_at, p.status
		FROM orders o
		JOIN shops s ON s.id = o.shop_id
		JOIN users u ON u.id = o.buyer_id
		LEFT JOIN payments p ON p.order_id = o.id
		WHERE `+whereClause+`
		ORDER BY o.created_at DESC
		LIMIT `+limitArg+` OFFSET `+offsetArg, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać zamówień")
		return
	}
	defer rows.Close()

	items := []OrderView{}
	for rows.Next() {
		var o OrderView
		var shippingRaw []byte
		var note *string
		var paymentStatus *string
		if err := rows.Scan(&o.ID, &o.ShopID, &o.ShopName, &o.BuyerUsername, &o.Status, &o.TotalAmount, &o.ShippingAmount, &o.Currency, &shippingRaw, &note, &o.CreatedAt, &paymentStatus); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu zamówień")
			return
		}
		json.Unmarshal(shippingRaw, &o.ShippingAddr)
		if note != nil {
			o.Note = *note
		}
		if paymentStatus != nil {
			o.PaymentStatus = *paymentStatus
		}
		items = append(items, o)
	}

	writeJSON(w, http.StatusOK, AdminOrdersResponse{
		Items: items, Total: total, Page: page, PageSize: pageSize,
		TotalPages: totalPagesOf(total, pageSize),
	})
}

func handleAdminListAuditLog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, pageSize := paginationParams(q)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var total int
	if err := dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM admin_audit_log`).Scan(&total); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się policzyć wpisów dziennika")
		return
	}

	rows, err := dbPool.Query(ctx, `
		SELECT a.id, a.admin_id, u.username, a.action, a.target_type, a.target_id, a.details, a.created_at
		FROM admin_audit_log a
		JOIN users u ON u.id = a.admin_id
		ORDER BY a.created_at DESC
		LIMIT $1 OFFSET $2
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać dziennika")
		return
	}
	defer rows.Close()

	items := []AdminAuditLogEntry{}
	for rows.Next() {
		var e AdminAuditLogEntry
		if err := rows.Scan(&e.ID, &e.AdminID, &e.AdminUsername, &e.Action, &e.TargetType, &e.TargetID, &e.Details, &e.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu dziennika")
			return
		}
		items = append(items, e)
	}

	writeJSON(w, http.StatusOK, AdminAuditLogResponse{
		Items: items, Total: total, Page: page, PageSize: pageSize,
		TotalPages: totalPagesOf(total, pageSize),
	})
}

func validateCategoryRequest(req *CategoryRequest) string {
	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 2 {
		return "Nazwa kategorii musi mieć min. 2 znaki"
	}
	return ""
}

func handleAdminListCategories(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := dbPool.Query(ctx, `
		SELECT id, parent_id, name, slug, description, sort_order
		FROM categories
		ORDER BY sort_order, name
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać kategorii")
		return
	}
	defer rows.Close()

	categories := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.Description, &c.SortOrder); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu kategorii")
			return
		}
		categories = append(categories, c)
	}

	writeJSON(w, http.StatusOK, categories)
}

func handleAdminCreateCategory(w http.ResponseWriter, r *http.Request) {
	adminID, _ := userIDFromContext(r.Context())

	var req CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	if msg := validateCategoryRequest(&req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if req.ParentID != nil && *req.ParentID != "" {
		var exists bool
		if err := dbPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)`, *req.ParentID).Scan(&exists); err != nil || !exists {
			writeError(w, http.StatusBadRequest, "Wybrana kategoria nadrzędna nie istnieje")
			return
		}
	} else {
		req.ParentID = nil
	}

	baseSlug := slugify(req.Name)
	if baseSlug == "" {
		baseSlug = "kategoria"
	}

	var c Category
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		slug := baseSlug
		if attempt > 0 {
			slug = fmt.Sprintf("%s-%d", baseSlug, attempt+1)
		}
		err = dbPool.QueryRow(ctx, `
			INSERT INTO categories (parent_id, name, slug, description, sort_order)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5)
			RETURNING id, parent_id, name, slug, description, sort_order
		`, req.ParentID, req.Name, slug, req.Description, req.SortOrder).Scan(
			&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.Description, &c.SortOrder,
		)
		if err == nil {
			break
		}
		if !isUniqueViolation(err) {
			writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć kategorii")
			return
		}
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć kategorii")
		return
	}

	logAdminAction(ctx, adminID, "category.create", "category", c.ID, c.Name)

	writeJSON(w, http.StatusCreated, c)
}

func handleAdminUpdateCategory(w http.ResponseWriter, r *http.Request) {
	adminID, _ := userIDFromContext(r.Context())
	id := r.PathValue("id")

	var req CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	if msg := validateCategoryRequest(&req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if req.ParentID != nil && *req.ParentID != "" {
		if *req.ParentID == id {
			writeError(w, http.StatusBadRequest, "Kategoria nie może być swoim własnym rodzicem")
			return
		}
		var exists bool
		if err := dbPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM categories WHERE id = $1)`, *req.ParentID).Scan(&exists); err != nil || !exists {
			writeError(w, http.StatusBadRequest, "Wybrana kategoria nadrzędna nie istnieje")
			return
		}
	} else {
		req.ParentID = nil
	}

	var c Category
	err := dbPool.QueryRow(ctx, `
		UPDATE categories
		SET parent_id = $1, name = $2, description = NULLIF($3, ''), sort_order = $4
		WHERE id = $5
		RETURNING id, parent_id, name, slug, description, sort_order
	`, req.ParentID, req.Name, req.Description, req.SortOrder, id).Scan(
		&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.Description, &c.SortOrder,
	)
	if err != nil {
		writeError(w, http.StatusNotFound, "Kategoria nie znaleziona")
		return
	}

	logAdminAction(ctx, adminID, "category.update", "category", c.ID, c.Name)

	writeJSON(w, http.StatusOK, c)
}

func handleAdminDeleteCategory(w http.ResponseWriter, r *http.Request) {
	adminID, _ := userIDFromContext(r.Context())
	id := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var name string
	if err := dbPool.QueryRow(ctx, `SELECT name FROM categories WHERE id = $1`, id).Scan(&name); err != nil {
		writeError(w, http.StatusNotFound, "Kategoria nie znaleziona")
		return
	}

	if _, err := dbPool.Exec(ctx, `UPDATE categories SET parent_id = NULL WHERE parent_id = $1`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się odłączyć podkategorii")
		return
	}

	tag, err := dbPool.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się usunąć kategorii")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Kategoria nie znaleziona")
		return
	}

	logAdminAction(ctx, adminID, "category.delete", "category", id, name)

	writeJSON(w, http.StatusOK, map[string]string{"message": "Kategoria usunięta"})
}

func totalPagesOf(total, pageSize int) int {
	tp := (total + pageSize - 1) / pageSize
	if tp < 1 {
		tp = 1
	}
	return tp
}
