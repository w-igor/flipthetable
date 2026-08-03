package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// wsHub manages all active WebSocket connections per user, enabling real-time push
// notifications (new messages, unread count updates) without requiring polling.
// Uses a map keyed by userID to track multiple connections per user.
type wsHub struct {
	mu      sync.Mutex
	clients map[string]map[*websocket.Conn]struct{}
}

// hub is the global WebSocket hub instance shared across all handlers.
var hub = &wsHub{clients: make(map[string]map[*websocket.Conn]struct{})}

// add registers a new WebSocket connection for a user, creating a new map if needed.
func (h *wsHub) add(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*websocket.Conn]struct{})
	}
	h.clients[userID][conn] = struct{}{}
}

// remove unregisters a WebSocket connection for a user, cleaning up empty user entries.
func (h *wsHub) remove(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.clients[userID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.clients, userID)
		}
	}
}

// sendTo broadcasts an event (as JSON) to all WebSocket connections for a given user.
// Silently skips if no connections exist or if sending fails (e.g., disconnected client).
func (h *wsHub) sendTo(userID string, event interface{}) {
	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.clients[userID]))
	for c := range h.clients[userID] {
		conns = append(conns, c)
	}
	h.mu.Unlock()

	if len(conns) == 0 {
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	// Send the event to each connection with a 5-second timeout
	for _, c := range conns {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		c.Write(ctx, websocket.MessageText, data)
		cancel()
	}
}

// handleWS upgrades an HTTP connection to WebSocket and holds it open for server-to-client
// push notifications. Authentication is done via JWT token in query parameter.
// The connection is one-way (server->client); incoming frames from the client are ignored.
func handleWS(w http.ResponseWriter, r *http.Request) {
	// Extract and validate JWT token from query string
	claims, err := parseToken(r.URL.Query().Get("token"), "access")
	if err != nil {
		http.Error(w, "Nieprawidłowy lub wygasły token", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	// Upgrade to WebSocket, allowing connections from any origin
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return
	}

	// Register connection and clean up on disconnect
	hub.add(userID, conn)
	defer func() {
		hub.remove(userID, conn)
		conn.CloseNow()
	}()

	// Block until the WebSocket connection closes (no-op read loop)
	ctx := conn.CloseRead(context.Background())
	<-ctx.Done()
}
