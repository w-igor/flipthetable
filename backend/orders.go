package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type listingLine struct {
	ShopID   string
	Price    string
	Quantity int
	Title    string
}

func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}

	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "Koszyk jest pusty")
		return
	}
	for _, item := range req.Items {
		if item.Quantity <= 0 {
			writeError(w, http.StatusBadRequest, "Nieprawidłowa ilość produktu")
			return
		}
	}
	if req.ShippingAddr.FullName == "" || req.ShippingAddr.Address == "" || req.ShippingAddr.City == "" {
		writeError(w, http.StatusBadRequest, "Uzupełnij dane wysyłki")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd bazy danych")
		return
	}
	defer tx.Rollback(ctx)

	listingIDs := make([]string, len(req.Items))
	for i, item := range req.Items {
		listingIDs[i] = item.ListingID
	}

	rows, err := tx.Query(ctx, `
		SELECT id, shop_id, price, quantity, title
		FROM listings
		WHERE id = ANY($1::uuid[]) AND is_active = TRUE
		FOR UPDATE
	`, listingIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się zweryfikować produktów")
		return
	}

	listings := map[string]listingLine{}
	for rows.Next() {
		var id string
		var l listingLine
		if err := rows.Scan(&id, &l.ShopID, &l.Price, &l.Quantity, &l.Title); err != nil {
			rows.Close()
			writeError(w, http.StatusInternalServerError, "Błąd odczytu produktów")
			return
		}
		listings[id] = l
	}
	rows.Close()

	shopGroups := map[string][]OrderItemRequest{}
	for _, item := range req.Items {
		l, found := listings[item.ListingID]
		if !found {
			writeError(w, http.StatusBadRequest, "Jeden z produktów już nie istnieje")
			return
		}
		if l.Quantity < item.Quantity {
			writeError(w, http.StatusConflict, "Za mało sztuk produktu \""+l.Title+"\" na stanie")
			return
		}
		shopGroups[l.ShopID] = append(shopGroups[l.ShopID], item)
	}

	shippingJSON, err := json.Marshal(req.ShippingAddr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd danych wysyłki")
		return
	}

	createdOrders := []OrderView{}

	for shopID, items := range shopGroups {
		var total float64
		for _, item := range items {
			l := listings[item.ListingID]
			total += parsePriceOrZero(l.Price) * float64(item.Quantity)
		}

		var orderID string
		var createdAt time.Time
		var shopName string
		err := tx.QueryRow(ctx, `
			INSERT INTO orders (buyer_id, shop_id, total_amount, shipping_addr, note)
			VALUES ($1, $2, $3, $4::jsonb, $5)
			RETURNING id, created_at
		`, userID, shopID, total, string(shippingJSON), req.Note).Scan(&orderID, &createdAt)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć zamówienia")
			return
		}

		if err := tx.QueryRow(ctx, `SELECT name FROM shops WHERE id = $1`, shopID).Scan(&shopName); err != nil {
			shopName = ""
		}

		orderItems := make([]OrderItemView, 0, len(items))
		for _, item := range items {
			l := listings[item.ListingID]
			var itemID string
			err := tx.QueryRow(ctx, `
				INSERT INTO order_items (order_id, listing_id, quantity, unit_price, title_snapshot)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`, orderID, item.ListingID, item.Quantity, l.Price, l.Title).Scan(&itemID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "Nie udało się utworzyć pozycji zamówienia")
				return
			}

			if _, err := tx.Exec(ctx, `UPDATE listings SET quantity = quantity - $1, sales_count = sales_count + $1 WHERE id = $2`, item.Quantity, item.ListingID); err != nil {
				writeError(w, http.StatusInternalServerError, "Nie udało się zaktualizować stanu magazynowego")
				return
			}

			orderItems = append(orderItems, OrderItemView{
				ID:            itemID,
				ListingID:     item.ListingID,
				Quantity:      item.Quantity,
				UnitPrice:     l.Price,
				TitleSnapshot: l.Title,
			})
		}

		createdOrders = append(createdOrders, OrderView{
			ID:          orderID,
			ShopID:      shopID,
			ShopName:    shopName,
			Status:      "pending",
			TotalAmount: formatPrice(total),
			Currency:    "PLN",
			ShippingAddr: req.ShippingAddr,
			Note:         req.Note,
			Items:        orderItems,
			CreatedAt:    createdAt,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się sfinalizować zamówienia")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{"orders": createdOrders})
}

func handleListOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := dbPool.Query(ctx, `
		SELECT o.id, o.shop_id, s.name, o.status, o.total_amount, o.currency,
		       o.shipping_addr, o.note, o.created_at
		FROM orders o
		JOIN shops s ON s.id = o.shop_id
		WHERE o.buyer_id = $1
		ORDER BY o.created_at DESC
	`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać zamówień")
		return
	}
	defer rows.Close()

	orders := []OrderView{}
	for rows.Next() {
		var o OrderView
		var shippingRaw []byte
		var note *string
		if err := rows.Scan(&o.ID, &o.ShopID, &o.ShopName, &o.Status, &o.TotalAmount, &o.Currency, &shippingRaw, &note, &o.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu zamówień")
			return
		}
		json.Unmarshal(shippingRaw, &o.ShippingAddr)
		if note != nil {
			o.Note = *note
		}
		orders = append(orders, o)
	}

	writeJSON(w, http.StatusOK, orders)
}

func handleGetOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}
	id := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var o OrderView
	var shippingRaw []byte
	var note *string
	err := dbPool.QueryRow(ctx, `
		SELECT o.id, o.shop_id, s.name, o.status, o.total_amount, o.currency,
		       o.shipping_addr, o.note, o.created_at
		FROM orders o
		JOIN shops s ON s.id = o.shop_id
		WHERE o.id = $1 AND o.buyer_id = $2
	`, id, userID).Scan(&o.ID, &o.ShopID, &o.ShopName, &o.Status, &o.TotalAmount, &o.Currency, &shippingRaw, &note, &o.CreatedAt)

	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "Zamówienie nie znalezione")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}
	json.Unmarshal(shippingRaw, &o.ShippingAddr)
	if note != nil {
		o.Note = *note
	}

	itemRows, err := dbPool.Query(ctx, `
		SELECT oi.id, oi.listing_id, oi.quantity, oi.unit_price, oi.title_snapshot,
		       (SELECT url FROM listing_photos p WHERE p.listing_id = oi.listing_id ORDER BY p.is_primary DESC, p.sort_order LIMIT 1)
		FROM order_items oi
		WHERE oi.order_id = $1
	`, id)
	if err == nil {
		defer itemRows.Close()
		for itemRows.Next() {
			var item OrderItemView
			if itemRows.Scan(&item.ID, &item.ListingID, &item.Quantity, &item.UnitPrice, &item.TitleSnapshot, &item.PhotoURL) == nil {
				o.Items = append(o.Items, item)
			}
		}
	}

	writeJSON(w, http.StatusOK, o)
}
