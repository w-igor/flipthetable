package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// wsHub tracks live WebSocket connections per user so events (new messages,
// unread-count changes) can be pushed instead of waiting on client polling.
type wsHub struct {
	mu      sync.Mutex
	clients map[string]map[*websocket.Conn]struct{}
}

var hub = &wsHub{clients: make(map[string]map[*websocket.Conn]struct{})}

func (h *wsHub) add(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*websocket.Conn]struct{})
	}
	h.clients[userID][conn] = struct{}{}
}

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

func (h *wsHub) sendTo(userID string, event any) {
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
	for _, c := range conns {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		c.Write(ctx, websocket.MessageText, data)
		cancel()
	}
}

// handleWS upgrades to a WebSocket and holds the connection open for
// server->client push only; the client never needs to send anything, so we
// just drain/discard any incoming frames until the socket closes.
func handleWS(w http.ResponseWriter, r *http.Request) {
	claims, err := parseToken(r.URL.Query().Get("token"), "access")
	if err != nil {
		http.Error(w, "Nieprawidłowy lub wygasły token", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return
	}

	hub.add(userID, conn)
	defer func() {
		hub.remove(userID, conn)
		conn.CloseNow()
	}()

	ctx := conn.CloseRead(context.Background())
	<-ctx.Done()
}
