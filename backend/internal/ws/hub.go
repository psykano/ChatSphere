package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/christopherjohns/chatsphere/internal/message"
	"nhooyr.io/websocket"
)

// Client represents a connected WebSocket user.
type Client struct {
	conn     *websocket.Conn
	userID   string
	username string
	roomID   string
	hub      *Hub
}

// Hub manages WebSocket clients grouped by room.
type Hub struct {
	mu      sync.RWMutex
	rooms   map[string]map[*Client]struct{}
	onJoin  func(roomID string, delta int)
}

// NewHub creates a new Hub. The onJoin callback is called with +1/-1
// when a client joins or leaves a room.
func NewHub(onJoin func(roomID string, delta int)) *Hub {
	return &Hub{
		rooms:  make(map[string]map[*Client]struct{}),
		onJoin: onJoin,
	}
}

// Envelope is the JSON structure sent over the WebSocket.
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// JoinPayload is sent by the client to join a room.
type JoinPayload struct {
	RoomID   string `json:"room_id"`
	Username string `json:"username"`
}

// ChatPayload is sent by the client to post a message.
type ChatPayload struct {
	Content string `json:"content"`
}

// addClient registers a client in its room.
func (h *Hub) addClient(c *Client) {
	h.mu.Lock()
	if h.rooms[c.roomID] == nil {
		h.rooms[c.roomID] = make(map[*Client]struct{})
	}
	h.rooms[c.roomID][c] = struct{}{}
	h.mu.Unlock()

	if h.onJoin != nil {
		h.onJoin(c.roomID, 1)
	}
}

// removeClient unregisters a client from its room.
func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	if clients, ok := h.rooms[c.roomID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.rooms, c.roomID)
		}
	}
	h.mu.Unlock()

	if h.onJoin != nil {
		h.onJoin(c.roomID, -1)
	}
}

// Broadcast sends a message to all clients in a room.
func (h *Hub) Broadcast(roomID string, msg *message.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws: failed to marshal message: %v", err)
		return
	}

	env := Envelope{Type: string(msg.Type), Payload: data}
	envData, err := json.Marshal(env)
	if err != nil {
		log.Printf("ws: failed to marshal envelope: %v", err)
		return
	}

	h.mu.RLock()
	clients := h.rooms[roomID]
	// Copy the set so we can release the lock before writing.
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.conn.Write(ctx, websocket.MessageText, envData); err != nil {
			log.Printf("ws: write to client %s failed: %v", c.userID, err)
		}
		cancel()
	}
}

// ClientCount returns the number of connected clients in a room.
func (h *Hub) ClientCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[roomID])
}
