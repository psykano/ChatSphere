package ws

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/christopherjohns/chatsphere/internal/message"
	"nhooyr.io/websocket"
)

// RoomValidator checks whether a room ID is valid and returns true if the
// client is allowed to join.
type RoomValidator func(roomID string) bool

// Handler handles WebSocket upgrade requests and client message loops.
type Handler struct {
	hub           *Hub
	validateRoom  RoomValidator
}

// NewHandler creates a new WebSocket Handler.
func NewHandler(hub *Hub, validateRoom RoomValidator) *Handler {
	return &Handler{
		hub:          hub,
		validateRoom: validateRoom,
	}
}

// ServeHTTP upgrades the HTTP connection to a WebSocket and runs the
// read loop for the client.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow all origins in dev; tighten in production.
	})
	if err != nil {
		log.Printf("ws: accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	client := &Client{
		conn:   conn,
		userID: generateClientID(),
		hub:    h.hub,
	}

	// First message must be a "join" envelope.
	if !h.handleJoin(r.Context(), client) {
		return
	}

	connCtx := h.hub.addClient(client)
	defer h.hub.removeClient(client)

	// Broadcast a system message that the user joined.
	h.hub.Broadcast(client.roomID, &message.Message{
		ID:        generateClientID(),
		RoomID:    client.roomID,
		Username:  client.username,
		Content:   client.username + " joined the room",
		Type:      message.TypeSystem,
		CreatedAt: time.Now(),
	})

	h.readLoop(r.Context(), connCtx, client)

	// Broadcast a system message that the user left.
	h.hub.Broadcast(client.roomID, &message.Message{
		ID:        generateClientID(),
		RoomID:    client.roomID,
		Username:  client.username,
		Content:   client.username + " left the room",
		Type:      message.TypeSystem,
		CreatedAt: time.Now(),
	})
}

// handleJoin reads the first message from the client and expects a "join"
// envelope. Returns true on success.
func (h *Handler) handleJoin(ctx context.Context, client *Client) bool {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, data, err := client.conn.Read(ctx)
	if err != nil {
		log.Printf("ws: read join error: %v", err)
		return false
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		closeWithError(client.conn, "invalid JSON")
		return false
	}
	if env.Type != "join" {
		closeWithError(client.conn, "first message must be type 'join'")
		return false
	}

	var payload JoinPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		closeWithError(client.conn, "invalid join payload")
		return false
	}
	if payload.RoomID == "" {
		closeWithError(client.conn, "room_id is required")
		return false
	}
	if payload.Username == "" {
		payload.Username = "anon-" + client.userID[:6]
	}

	if h.validateRoom != nil && !h.validateRoom(payload.RoomID) {
		closeWithError(client.conn, "room not found")
		return false
	}

	client.roomID = payload.RoomID
	client.username = payload.Username
	return true
}

// readLoop reads messages from the client until the connection closes
// or the connection manager cancels connCtx.
func (h *Handler) readLoop(ctx context.Context, connCtx context.Context, client *Client) {
	for {
		select {
		case <-connCtx.Done():
			return
		default:
		}

		_, data, err := client.conn.Read(ctx)
		if err != nil {
			// Normal close or context cancelled.
			return
		}

		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}

		switch env.Type {
		case "chat":
			var payload ChatPayload
			if err := json.Unmarshal(env.Payload, &payload); err != nil || payload.Content == "" {
				continue
			}
			h.hub.Broadcast(client.roomID, &message.Message{
				ID:        generateClientID(),
				RoomID:    client.roomID,
				UserID:    client.userID,
				Username:  client.username,
				Content:   payload.Content,
				Type:      message.TypeChat,
				CreatedAt: time.Now(),
			})
		}
	}
}

func closeWithError(conn *websocket.Conn, reason string) {
	conn.Close(websocket.StatusPolicyViolation, reason)
}

func generateClientID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
