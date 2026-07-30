package main

import "time"

type Message struct {
	ID         string    `json:"id"`
	SenderID   string    `json:"sender_id"`
	ReceiverID string    `json:"receiver_id"`
	OrderID    *string   `json:"order_id,omitempty"`
	Body       string    `json:"body"`
	IsRead     bool      `json:"is_read"`
	SentAt     time.Time `json:"sent_at"`
}

type SendMessageRequest struct {
	ReceiverID string  `json:"receiver_id"`
	OrderID    *string `json:"order_id,omitempty"`
	Body       string  `json:"body"`
}

type Conversation struct {
	UserID      string    `json:"user_id"`
	Username    string    `json:"username"`
	LastBody    string    `json:"last_body"`
	LastSentAt  time.Time `json:"last_sent_at"`
	UnreadCount int       `json:"unread_count"`
}
