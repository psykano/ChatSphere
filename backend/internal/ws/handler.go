package ws

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/christopherjohns/chatsphere/internal/message"
	"nhooyr.io/websocket"
)

// RoomValidator checks whether a room ID is valid and the client is allowed
// to join. It returns an empty string on success or an error reason on failure.
type RoomValidator func(roomID string) string

// Handler handles WebSocket upgrade requests and client message loops.
type Handler struct {
	hub          *Hub
	validateRoom RoomValidator
	sessions     *SessionStore
	messages     message.MessageStore
}

// NewHandler creates a new WebSocket Handler.
func NewHandler(hub *Hub, validateRoom RoomValidator, sessions *SessionStore, messages message.MessageStore) *Handler {
	return &Handler{
		hub:          hub,
		validateRoom: validateRoom,
		sessions:     sessions,
		messages:     messages,
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
	defer func() {
		h.hub.removeClient(client)
		h.sessions.MarkDisconnected(client.sessionID)
	}()

	// Only broadcast "joined" for new connections, not resumptions.
	if !client.resumed {
		h.hub.Broadcast(client.roomID, &message.Message{
			ID:        generateClientID(),
			RoomID:    client.roomID,
			Username:  client.username,
			Content:   client.username + " joined the room",
			Type:      message.TypeSystem,
			CreatedAt: time.Now(),
		})
	}

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
// envelope. It supports session resumption via session_id in the payload.
// Returns true on success, and sets client.resumed if the session was resumed.
func (h *Handler) handleJoin(ctx context.Context, client *Client) bool {
	joinCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, data, err := client.conn.Read(joinCtx)
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

	if h.validateRoom != nil {
		if reason := h.validateRoom(payload.RoomID); reason != "" {
			closeWithError(client.conn, reason)
			return false
		}
	}

	// Attempt session resumption.
	resumed := false
	if payload.SessionID != "" {
		if sess := h.sessions.Get(payload.SessionID); sess != nil && !sess.connected() && sess.RoomID == payload.RoomID {
			client.userID = sess.UserID
			client.username = sess.Username
			client.sessionID = sess.ID
			h.sessions.MarkConnected(sess.ID)
			resumed = true
		}
	}

	if !resumed {
		if payload.Username == "" {
			payload.Username = "anon-" + client.userID[:6]
		}
		client.roomID = payload.RoomID
		client.username = payload.Username
		sess := h.sessions.Create(client.userID, client.username, client.roomID)
		client.sessionID = sess.ID
	} else {
		client.roomID = payload.RoomID
	}

	client.resumed = resumed

	// Send session info back to client.
	h.sendSessionInfo(ctx, client, resumed)

	// Send missed messages on session resumption, or recent history for new joins.
	if resumed {
		h.sendBackfill(ctx, client)
	} else {
		h.sendHistory(ctx, client)
	}

	return true
}

// sendSessionInfo writes the session envelope to the client.
func (h *Handler) sendSessionInfo(ctx context.Context, client *Client, resumed bool) {
	sp := SessionPayload{
		SessionID: client.sessionID,
		UserID:    client.userID,
		Username:  client.username,
		Resumed:   resumed,
	}
	data, err := json.Marshal(sp)
	if err != nil {
		log.Printf("ws: failed to marshal session payload: %v", err)
		return
	}
	env, err := json.Marshal(Envelope{Type: "session", Payload: data})
	if err != nil {
		log.Printf("ws: failed to marshal session envelope: %v", err)
		return
	}
	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	if err := client.conn.Write(writeCtx, websocket.MessageText, env); err != nil {
		log.Printf("ws: failed to write session info: %v", err)
	}
}

// sendBackfill sends missed messages to a client that is resuming a session.
func (h *Handler) sendBackfill(ctx context.Context, client *Client) {
	if h.messages == nil {
		return
	}

	sess := h.sessions.Get(client.sessionID)
	if sess == nil {
		return
	}

	missed := h.messages.After(client.roomID, sess.LastMessageID)
	if len(missed) == 0 {
		return
	}

	data, err := json.Marshal(missed)
	if err != nil {
		log.Printf("ws: failed to marshal backfill: %v", err)
		return
	}

	env, err := json.Marshal(Envelope{Type: "backfill", Payload: data})
	if err != nil {
		log.Printf("ws: failed to marshal backfill envelope: %v", err)
		return
	}

	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	if err := client.conn.Write(writeCtx, websocket.MessageText, env); err != nil {
		log.Printf("ws: failed to write backfill: %v", err)
	}

	// Update last message ID to the last backfilled message.
	last := missed[len(missed)-1]
	h.sessions.SetLastMessageID(client.sessionID, last.ID)
}

// historyLimit is the number of recent messages to send on room join.
const historyLimit = 50

// sendHistory sends recent message history to a newly joined client.
// An empty history envelope is always sent so clients can rely on
// receiving it as part of the join handshake.
func (h *Handler) sendHistory(ctx context.Context, client *Client) {
	var recent []*message.Message
	if h.messages != nil {
		recent = h.messages.Recent(client.roomID, historyLimit)
	}
	if recent == nil {
		recent = []*message.Message{}
	}

	data, err := json.Marshal(recent)
	if err != nil {
		log.Printf("ws: failed to marshal history: %v", err)
		return
	}

	env, err := json.Marshal(Envelope{Type: "history", Payload: data})
	if err != nil {
		log.Printf("ws: failed to marshal history envelope: %v", err)
		return
	}

	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	if err := client.conn.Write(writeCtx, websocket.MessageText, env); err != nil {
		log.Printf("ws: failed to write history: %v", err)
	}
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

		// Mark activity so idle reaping doesn't close active connections.
		h.hub.ConnMgr().TouchActivity(client)

		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}

		switch env.Type {
		case "chat":
			var payload ChatPayload
			if err := json.Unmarshal(env.Payload, &payload); err != nil {
				continue
			}
			content := strings.TrimSpace(payload.Content)
			if content == "" {
				h.sendError(ctx, client, "message content is required")
				continue
			}
			if len(content) > maxMessageLength {
				h.sendError(ctx, client, "message exceeds maximum length of 2000 characters")
				continue
			}
			h.hub.Broadcast(client.roomID, &message.Message{
				ID:        generateClientID(),
				RoomID:    client.roomID,
				UserID:    client.userID,
				Username:  client.username,
				Content:   content,
				Type:      message.TypeChat,
				CreatedAt: time.Now(),
			})
		}
	}
}

// sendError writes an error envelope to the client.
func (h *Handler) sendError(ctx context.Context, client *Client, msg string) {
	data, err := json.Marshal(ErrorPayload{Message: msg})
	if err != nil {
		return
	}
	env, err := json.Marshal(Envelope{Type: "error", Payload: data})
	if err != nil {
		return
	}
	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	if err := client.conn.Write(writeCtx, websocket.MessageText, env); err != nil {
		log.Printf("ws: failed to write error to client %s: %v", client.userID, err)
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
