package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketHub struct {
	clients    map[string][]*Client
	broadcast  chan interface{}
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type Client struct {
	userID   int
	email    string
	conn     *websocket.Conn
	send     chan interface{}
	hub      *WebSocketHub
}

type WSMessage struct {
	Type      string      `json:"type"`
	UserID    int         `json:"user_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type Notification struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Type      string    `json:"type"` // "order_status", "order_created", "order_shipped", etc
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	OrderID   int       `json:"order_id,omitempty"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

var hub *WebSocketHub

func initWebSocketHub() {
	hub = &WebSocketHub{
		clients:    make(map[string][]*Client),
		broadcast:  make(chan interface{}, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	go hub.run()
}

func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.email] = append(h.clients[client.email], client)
			h.mu.Unlock()
			log.Printf("✓ Client connected: %s (total: %d)", client.email, len(h.clients[client.email]))

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.email]; ok {
				for i, c := range clients {
					if c == client {
						h.clients[client.email] = append(clients[:i], clients[i+1:]...)
						break
					}
				}
				if len(h.clients[client.email]) == 0 {
					delete(h.clients, client.email)
				}
			}
			h.mu.Unlock()
			close(client.send)
			log.Printf("✗ Client disconnected: %s", client.email)

		case msg := <-h.broadcast:
			if wsMsg, ok := msg.(WSMessage); ok {
				h.mu.RLock()
				if clients, ok := h.clients[wsMsg.Data.(map[string]interface{})["email"]]; ok {
					for _, client := range clients {
						select {
						case client.send <- wsMsg:
						default:
							// Channel full, skip
						}
					}
				}
				h.mu.RUnlock()
			}
		}
	}
}

func (h *WebSocketHub) sendToUser(email string, msg WSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[email]; ok {
		for _, client := range clients {
			select {
			case client.send <- msg:
			default:
				// Channel full, skip
			}
		}
	}
}

func (h *WebSocketHub) broadcastToUser(email string, notif Notification) {
	msg := WSMessage{
		Type:      notif.Type,
		Timestamp: time.Now(),
		Data:      notif,
	}
	h.sendToUser(email, msg)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local dev
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	email := r.Header.Get("X-Email")
	if email == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		email: email,
		conn:  conn,
		send:  make(chan interface{}, 256),
		hub:   hub,
	}

	hub.register <- client

	go client.readPump()
	go client.writePump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		msg := WSMessage{}
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		// Handle ping/pong
		if msg.Type == "ping" {
			c.send <- WSMessage{
				Type:      "pong",
				Timestamp: time.Now(),
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Helper functions to send notifications

func notifyOrderCreated(userEmail string, orderID int, totalPrice float64) {
	notif := Notification{
		UserID:    0, // Would be set in real scenario
		Type:      "order_created",
		Title:     "Zamówienie złożone",
		Message:   "Twoje zamówienie zostało złożone",
		OrderID:   orderID,
		CreatedAt: time.Now(),
	}
	hub.broadcastToUser(userEmail, notif)

	// Save to DB
	db.Exec(
		"INSERT INTO notifications (user_email, type, title, message, order_id) VALUES ($1, $2, $3, $4, $5)",
		userEmail, notif.Type, notif.Title, notif.Message, orderID,
	)
}

func notifyOrderStatusChange(userEmail string, orderID int, newStatus string) {
	statusLabels := map[string]string{
		"pending":    "Oczekujące",
		"confirmed":  "Potwierdzone",
		"shipped":    "Wysłane",
		"delivered":  "Dostarczone",
		"cancelled":  "Anulowane",
	}

	notif := Notification{
		Type:      "order_status",
		Title:     "Status zamówienia",
		Message:   "Zamówienie " + statusLabels[newStatus],
		OrderID:   orderID,
		CreatedAt: time.Now(),
	}
	hub.broadcastToUser(userEmail, notif)

	// Save to DB
	db.Exec(
		"INSERT INTO notifications (user_email, type, title, message, order_id) VALUES ($1, $2, $3, $4, $5)",
		userEmail, notif.Type, notif.Title, notif.Message, orderID,
	)
}
