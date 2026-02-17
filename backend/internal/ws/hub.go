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
	isCreator bool
	kicked    bool // set when the user is kicked/banned to suppress "left" message
}

// Hub manages WebSocket clients grouped by room.
type Hub struct {
	mu          sync.RWMutex
	rooms       map[string]map[*Client]struct{}
	hosts       map[string]string               // roomID → host userID
	banned      map[string]map[string]struct{}   // roomID → set of banned userIDs
	muted       map[string]map[string]struct{}   // roomID → set of muted userIDs
	conns       *ConnManager
	messages    message.MessageStore
	sessions    *SessionStore
	onJoin      func(roomID string, delta int)
	onBroadcast func(roomID string)
}

// NewHub creates a new Hub. The onJoin callback is called with +1/-1
// when a client joins or leaves a room.
func NewHub(onJoin func(roomID string, delta int)) *Hub {
	return &Hub{
		rooms:  make(map[string]map[*Client]struct{}),
		hosts:  make(map[string]string),
		banned: make(map[string]map[string]struct{}),
		muted:  make(map[string]map[string]struct{}),
		conns:  NewConnManager(),
		onJoin: onJoin,
	}
}

// SetMessageStore sets the message store used for backfill on reconnect.
func (h *Hub) SetMessageStore(store message.MessageStore) {
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

// HistoryFetchPayload is sent by the client to request older messages.
type HistoryFetchPayload struct {
	BeforeID string `json:"before_id"`
	Limit    int    `json:"limit"`
}

// HistoryBatchPayload is sent by the server with a batch of older messages.
type HistoryBatchPayload struct {
	Messages []*message.Message `json:"messages"`
	HasMore  bool               `json:"has_more"`
}

// ErrorPayload is sent by the server when a client message is rejected.
type ErrorPayload struct {
	Message string `json:"message"`
}

// KickPayload is sent by a room creator to kick a user.
type KickPayload struct {
	UserID string `json:"user_id"`
}

// BanPayload is sent by a room creator to ban a user.
type BanPayload struct {
	UserID string `json:"user_id"`
}

// MutePayload is sent by a room creator to mute/unmute a user.
type MutePayload struct {
	UserID string `json:"user_id"`
}

// SetUsernamePayload is sent by the client to change their username in the room.
type SetUsernamePayload struct {
	Username string `json:"username"`
}

// TypingPayload is broadcast by the server to indicate a user is typing.
type TypingPayload struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// maxMessageLength is the maximum allowed length for a chat message.
const maxMessageLength = 2000

// maxUsernameLength is the maximum allowed length for a username.
const maxUsernameLength = 30

// addClient registers a client in its room and starts its write pump.
// Returns a context that is cancelled when the client is removed.
func (h *Hub) addClient(c *Client) context.Context {
	ctx := h.conns.Add(c)

	h.mu.Lock()
	if h.rooms[c.roomID] == nil {
		h.rooms[c.roomID] = make(map[*Client]struct{})
	}
	h.rooms[c.roomID][c] = struct{}{}
	// First client to join becomes the room host.
	if _, ok := h.hosts[c.roomID]; !ok {
		h.hosts[c.roomID] = c.userID
		c.isCreator = true
	} else if h.hosts[c.roomID] == c.userID {
		c.isCreator = true
	}
	h.mu.Unlock()

	if h.onJoin != nil {
		h.onJoin(c.roomID, 1)
	}
	return ctx
}

// removeClient unregisters a client from its room and stops its write pump.
// It is safe to call multiple times (e.g. after KickClient).
func (h *Hub) removeClient(c *Client) {
	h.conns.Remove(c)

	removed := false
	h.mu.Lock()
	if clients, ok := h.rooms[c.roomID]; ok {
		if _, exists := clients[c]; exists {
			delete(clients, c)
			removed = true
			if len(clients) == 0 {
				delete(h.rooms, c.roomID)
			}
		}
	}
	h.mu.Unlock()

	if removed && h.onJoin != nil {
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

// BroadcastEphemeral sends a message to all clients in a room except the
// sender. Unlike Broadcast, it does not persist the message or update session
// tracking. This is intended for transient signals like typing indicators.
func (h *Hub) BroadcastEphemeral(roomID string, sender *Client, msg *message.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws: failed to marshal ephemeral message: %v", err)
		return
	}

	env := Envelope{Type: string(msg.Type), Payload: data}
	envData, err := json.Marshal(env)
	if err != nil {
		log.Printf("ws: failed to marshal ephemeral envelope: %v", err)
		return
	}

	h.mu.RLock()
	clients := h.rooms[roomID]
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		if c != sender {
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range targets {
		h.conns.Send(c, envData)
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
	delete(h.hosts, roomID)
	delete(h.banned, roomID)
	delete(h.muted, roomID)
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

// FindClient returns the client with the given userID in the room, or nil.
func (h *Hub) FindClient(roomID, userID string) *Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[roomID] {
		if c.userID == userID {
			return c
		}
	}
	return nil
}

// IsBanned returns true if the user is banned from the room.
func (h *Hub) IsBanned(roomID, userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.banned[roomID][userID]
	return ok
}

// Ban adds a user to the room's ban list.
func (h *Hub) Ban(roomID, userID string) {
	h.mu.Lock()
	if h.banned[roomID] == nil {
		h.banned[roomID] = make(map[string]struct{})
	}
	h.banned[roomID][userID] = struct{}{}
	h.mu.Unlock()
}

// IsMuted returns true if the user is muted in the room.
func (h *Hub) IsMuted(roomID, userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.muted[roomID][userID]
	return ok
}

// Mute toggles a user's mute status in the room. Returns true if the user
// is muted after the call.
func (h *Hub) Mute(roomID, userID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.muted[roomID] == nil {
		h.muted[roomID] = make(map[string]struct{})
	}
	if _, ok := h.muted[roomID][userID]; ok {
		delete(h.muted[roomID], userID)
		return false
	}
	h.muted[roomID][userID] = struct{}{}
	return true
}

// KickClient forcefully disconnects a client from its room.
// It removes the client from the room map first to prevent
// subsequent broadcasts from sending to a closed channel.
func (h *Hub) KickClient(c *Client) {
	c.kicked = true

	h.mu.Lock()
	if clients, ok := h.rooms[c.roomID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.rooms, c.roomID)
		}
	}
	h.mu.Unlock()

	h.conns.Remove(c)

	if h.onJoin != nil {
		h.onJoin(c.roomID, -1)
	}
}
