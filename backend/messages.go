package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	var req SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Nieprawidłowe dane wejściowe")
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.ReceiverID == "" || req.Body == "" {
		writeError(w, http.StatusBadRequest, "Wiadomość nie może być pusta")
		return
	}
	if req.ReceiverID == userID {
		writeError(w, http.StatusBadRequest, "Nie możesz wysłać wiadomości do samego siebie")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var receiverExists bool
	if err := dbPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, req.ReceiverID).Scan(&receiverExists); err != nil {
		writeError(w, http.StatusInternalServerError, "Błąd serwera")
		return
	}
	if !receiverExists {
		writeError(w, http.StatusBadRequest, "Odbiorca nie istnieje")
		return
	}

	var msg Message
	err := dbPool.QueryRow(ctx, `
		INSERT INTO messages (sender_id, receiver_id, order_id, body)
		VALUES ($1, $2, $3, $4)
		RETURNING id, sender_id, receiver_id, order_id, body, is_read, sent_at
	`, userID, req.ReceiverID, req.OrderID, req.Body).Scan(
		&msg.ID, &msg.SenderID, &msg.ReceiverID, &msg.OrderID, &msg.Body, &msg.IsRead, &msg.SentAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się wysłać wiadomości")
		return
	}

	hub.sendTo(req.ReceiverID, map[string]any{"type": "message", "data": msg})
	var unread int
	if err := dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM messages WHERE receiver_id = $1 AND is_read = FALSE`, req.ReceiverID).Scan(&unread); err == nil {
		hub.sendTo(req.ReceiverID, map[string]any{"type": "unread_count", "count": unread})
	}

	writeJSON(w, http.StatusCreated, msg)
}

func handleListConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := dbPool.Query(ctx, `
		WITH convo AS (
			SELECT
				CASE WHEN sender_id = $1 THEN receiver_id ELSE sender_id END AS other_id,
				body, sent_at
			FROM messages
			WHERE sender_id = $1 OR receiver_id = $1
		),
		ranked AS (
			SELECT other_id, body, sent_at,
			       ROW_NUMBER() OVER (PARTITION BY other_id ORDER BY sent_at DESC) AS rn
			FROM convo
		)
		SELECT r.other_id, u.username, r.body, r.sent_at,
		       (SELECT COUNT(*) FROM messages m WHERE m.sender_id = r.other_id AND m.receiver_id = $1 AND m.is_read = FALSE)
		FROM ranked r
		JOIN users u ON u.id = r.other_id
		WHERE r.rn = 1
		ORDER BY r.sent_at DESC
	`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać wiadomości")
		return
	}
	defer rows.Close()

	conversations := []Conversation{}
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.UserID, &c.Username, &c.LastBody, &c.LastSentAt, &c.UnreadCount); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu wiadomości")
			return
		}
		conversations = append(conversations, c)
	}

	writeJSON(w, http.StatusOK, conversations)
}

func handleUnreadMessageCount(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var count int
	if err := dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM messages WHERE receiver_id = $1 AND is_read = FALSE`, userID).Scan(&count); err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać liczby wiadomości")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

func handleGetThread(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "Brak autoryzacji")
		return
	}
	otherID := r.PathValue("userId")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := dbPool.Query(ctx, `
		SELECT id, sender_id, receiver_id, order_id, body, is_read, sent_at
		FROM messages
		WHERE (sender_id = $1 AND receiver_id = $2) OR (sender_id = $2 AND receiver_id = $1)
		ORDER BY sent_at ASC
	`, userID, otherID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się pobrać wiadomości")
		return
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.SenderID, &m.ReceiverID, &m.OrderID, &m.Body, &m.IsRead, &m.SentAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Błąd odczytu wiadomości")
			return
		}
		messages = append(messages, m)
	}

	tag, err := dbPool.Exec(ctx, `
		UPDATE messages SET is_read = TRUE WHERE sender_id = $1 AND receiver_id = $2 AND is_read = FALSE
	`, otherID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nie udało się oznaczyć wiadomości jako przeczytanych")
		return
	}
	if tag.RowsAffected() > 0 {
		var unread int
		if err := dbPool.QueryRow(ctx, `SELECT COUNT(*) FROM messages WHERE receiver_id = $1 AND is_read = FALSE`, userID).Scan(&unread); err == nil {
			hub.sendTo(userID, map[string]any{"type": "unread_count", "count": unread})
		}
	}

	writeJSON(w, http.StatusOK, messages)
}
