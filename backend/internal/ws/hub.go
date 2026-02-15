package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/christopherjohns/chatsphere/internal/message"
	"nhooyr.io/websocket"
)

// Client represents a connected WebSocket user.
type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	userID    string
	username  string
	roomID    string
	sessionID string
	resumed   bool
	hub       *Hub
}

// Hub manages WebSocket clients grouped by room.
type Hub struct {
	mu          sync.RWMutex
	rooms       map[string]map[*Client]struct{}
	conns       *ConnManager
	messages    *message.Store
	sessions    *SessionStore
	onJoin      func(roomID string, delta int)
	onBroadcast func(roomID string)
}

// NewHub creates a new Hub. The onJoin callback is called with +1/-1
// when a client joins or leaves a room.
func NewHub(onJoin func(roomID string, delta int)) *Hub {
	return &Hub{
		rooms:  make(map[string]map[*Client]struct{}),
		conns:  NewConnManager(),
		onJoin: onJoin,
	}
}

// SetMessageStore sets the message store used for backfill on reconnect.
func (h *Hub) SetMessageStore(store *message.Store) {
	h.messages = store
}

// SetSessionStore sets the session store used to track last message IDs.
func (h *Hub) SetSessionStore(sessions *SessionStore) {
	h.sessions = sessions
}

// SetOnBroadcast sets a callback invoked after each broadcast for a room.
func (h *Hub) SetOnBroadcast(fn func(roomID string)) {
	h.onBroadcast = fn
}

// ConnMgr returns the connection manager for this hub.
func (h *Hub) ConnMgr() *ConnManager {
	return h.conns
}

// Envelope is the JSON structure sent over the WebSocket.
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// JoinPayload is sent by the client to join a room.
type JoinPayload struct {
	RoomID    string `json:"room_id"`
	Username  string `json:"username"`
	SessionID string `json:"session_id,omitempty"`
}

// SessionPayload is sent by the server after a successful join or resume.
type SessionPayload struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Resumed   bool   `json:"resumed"`
}

// ChatPayload is sent by the client to post a message.
type ChatPayload struct {
	Content string `json:"content"`
}

// ErrorPayload is sent by the server when a client message is rejected.
type ErrorPayload struct {
	Message string `json:"message"`
}

// maxMessageLength is the maximum allowed length for a chat message.
const maxMessageLength = 2000

// addClient registers a client in its room and starts its write pump.
// Returns a context that is cancelled when the client is removed.
func (h *Hub) addClient(c *Client) context.Context {
	ctx := h.conns.Add(c)

	h.mu.Lock()
	if h.rooms[c.roomID] == nil {
		h.rooms[c.roomID] = make(map[*Client]struct{})
	}
	h.rooms[c.roomID][c] = struct{}{}
	h.mu.Unlock()

	if h.onJoin != nil {
		h.onJoin(c.roomID, 1)
	}
	return ctx
}

// removeClient unregisters a client from its room and stops its write pump.
func (h *Hub) removeClient(c *Client) {
	h.conns.Remove(c)

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

// Broadcast sends a message to all clients in a room and persists it
// to the message store for backfill on reconnect.
func (h *Hub) Broadcast(roomID string, msg *message.Message) {
	if h.messages != nil {
		h.messages.Append(msg)
	}

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
	// Copy the set so we can release the lock before sending.
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		if h.conns.Send(c, envData) && h.sessions != nil {
			h.sessions.SetLastMessageID(c.sessionID, msg.ID)
		}
	}

	if h.onBroadcast != nil {
		h.onBroadcast(roomID)
	}
}

// DisconnectRoom closes all client connections in a room and removes them
// from the hub. The onJoin callback is NOT fired for these removals since
// the room is being expired.
func (h *Hub) DisconnectRoom(roomID string) {
	h.mu.Lock()
	clients := h.rooms[roomID]
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		targets = append(targets, c)
	}
	delete(h.rooms, roomID)
	h.mu.Unlock()

	for _, c := range targets {
		h.conns.Remove(c)
	}
}

// ClientCount returns the number of connected clients in a room.
func (h *Hub) ClientCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[roomID])
}
