// Package stream provides a WebSocket fan-out hub for live operator updates.
package stream

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// MessageType identifies stream payloads sent to dashboards.
type MessageType string

const (
	TypeEvent    MessageType = "event"
	TypeDigest   MessageType = "digest"
	TypeDecision MessageType = "decision"
	TypeAlert    MessageType = "alert"
)

// Message is a JSON envelope on /api/v1/stream.
type Message struct {
	Type MessageType     `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Hub broadcasts messages to connected WebSocket clients.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

// NewHub returns an empty hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[*websocket.Conn]struct{})}
}

// Register adds a client connection.
func (h *Hub) Register(c *websocket.Conn) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

// Unregister removes a client connection.
func (h *Hub) Unregister(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	_ = c.Close()
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// Broadcast sends a message to all clients; slow clients are dropped.
func (h *Hub) Broadcast(msg Message) {
	body, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if err := c.WriteMessage(websocket.TextMessage, body); err != nil {
			go h.drop(c)
		}
	}
}

// BroadcastJSON marshals v and sends it with the given type.
func (h *Hub) BroadcastJSON(typ MessageType, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		return
	}
	h.Broadcast(Message{Type: typ, Data: raw})
}

func (h *Hub) drop(c *websocket.Conn) {
	h.Unregister(c)
}
