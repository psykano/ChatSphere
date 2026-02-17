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
	"github.com/christopherjohns/chatsphere/internal/ratelimit"
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
	chatLimiter  *ratelimit.IPLimiter
}

// NewHandler creates a new WebSocket Handler.
func NewHandler(hub *Hub, validateRoom RoomValidator, sessions *SessionStore, messages message.MessageStore) *Handler {
	return &Handler{
		hub:          hub,
		validateRoom: validateRoom,
		sessions:     sessions,
		messages:     messages,
		chatLimiter:  ratelimit.NewIPLimiter(10, 10*time.Second),
	}
}

// SetChatLimiter replaces the default chat rate limiter (for testing).
func (h *Handler) SetChatLimiter(l *ratelimit.IPLimiter) {
	h.chatLimiter = l
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
			Action:    message.ActionJoin,
			CreatedAt: time.Now(),
		})
	}

	h.readLoop(r.Context(), connCtx, client)

	// Broadcast a "left" message unless the user was kicked/banned
	// (those actions already broadcast their own system message).
	if !client.kicked {
		h.hub.Broadcast(client.roomID, &message.Message{
			ID:        generateClientID(),
			RoomID:    client.roomID,
			Username:  client.username,
			Content:   client.username + " left the room",
			Type:      message.TypeSystem,
			Action:    message.ActionLeave,
			CreatedAt: time.Now(),
		})
	}
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

	// Check if the user is banned from this room (by session).
	if payload.SessionID != "" {
		if sess := h.sessions.Get(payload.SessionID); sess != nil {
			if h.hub.IsBanned(payload.RoomID, sess.UserID) {
				closeWithError(client.conn, "you are banned from this room")
				return false
			}
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
		payload.Username = strings.TrimSpace(payload.Username)
		if payload.Username == "" {
			payload.Username = "anon-" + client.userID[:6]
		}
		if len(payload.Username) > maxUsernameLength {
			closeWithError(client.conn, "username must be 30 characters or less")
			return false
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

// BackfillPayload wraps the missed messages with metadata about gaps.
type BackfillPayload struct {
	Messages []*message.Message `json:"messages"`
	HasGap   bool               `json:"has_gap"`
}

// sendBackfill sends missed messages to a client that is resuming a session.
// If the last message ID was evicted from the store, it falls back to recent
// messages and sets has_gap to true so the client can show a gap indicator.
func (h *Handler) sendBackfill(ctx context.Context, client *Client) {
	if h.messages == nil {
		return
	}

	sess := h.sessions.Get(client.sessionID)
	if sess == nil {
		return
	}

	missed := h.messages.After(client.roomID, sess.LastMessageID)
	hasGap := false

	// If After() returned nil but the room has messages, the LastMessageID
	// was evicted from the store. Fall back to recent messages.
	if missed == nil && sess.LastMessageID != "" && h.messages.Count(client.roomID) > 0 {
		missed = h.messages.Recent(client.roomID, backfillLimit)
		hasGap = true
	}

	if len(missed) == 0 {
		return
	}

	// Cap the number of backfilled messages.
	if len(missed) > backfillLimit {
		missed = missed[len(missed)-backfillLimit:]
		hasGap = true
	}

	payload := BackfillPayload{
		Messages: missed,
		HasGap:   hasGap,
	}

	data, err := json.Marshal(payload)
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

// backfillLimit caps how many missed messages to send on reconnect.
const backfillLimit = 200

// historyBatchDefault is the default number of older messages per batch.
const historyBatchDefault = 50

// historyBatchMax caps the number of older messages per batch.
const historyBatchMax = 100

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

// sendHistoryBatch sends a batch of older messages to a client that requested them.
func (h *Handler) sendHistoryBatch(ctx context.Context, client *Client, req HistoryFetchPayload) {
	if h.messages == nil || req.BeforeID == "" {
		return
	}

	limit := req.Limit
	if limit <= 0 {
		limit = historyBatchDefault
	}
	if limit > historyBatchMax {
		limit = historyBatchMax
	}

	// Fetch one extra to detect if more messages exist.
	msgs := h.messages.Before(client.roomID, req.BeforeID, limit+1)

	hasMore := false
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
		hasMore = true
	}

	if msgs == nil {
		msgs = []*message.Message{}
	}

	payload := HistoryBatchPayload{
		Messages: msgs,
		HasMore:  hasMore,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("ws: failed to marshal history batch: %v", err)
		return
	}

	env, err := json.Marshal(Envelope{Type: "history_batch", Payload: data})
	if err != nil {
		log.Printf("ws: failed to marshal history batch envelope: %v", err)
		return
	}

	writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	if err := client.conn.Write(writeCtx, websocket.MessageText, env); err != nil {
		log.Printf("ws: failed to write history batch: %v", err)
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
			if h.hub.IsMuted(client.roomID, client.userID) {
				h.sendError(ctx, client, "you are muted in this room")
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
			if !h.chatLimiter.Allow(client.userID) {
				h.sendError(ctx, client, "rate limit exceeded: max 10 messages per 10 seconds")
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
		case "kick":
			h.handleKick(ctx, client, env.Payload)
		case "ban":
			h.handleBan(ctx, client, env.Payload)
		case "mute":
			h.handleMute(ctx, client, env.Payload)
		case "history_fetch":
			var payload HistoryFetchPayload
			if err := json.Unmarshal(env.Payload, &payload); err != nil {
				continue
			}
			h.sendHistoryBatch(ctx, client, payload)
		case "typing":
			h.hub.BroadcastEphemeral(client.roomID, client, &message.Message{
				RoomID:   client.roomID,
				UserID:   client.userID,
				Username: client.username,
				Type:     message.TypeTyping,
			})
		case "leave":
			return
		}
	}
}

// handleKick removes a user from the room.
func (h *Handler) handleKick(ctx context.Context, client *Client, payload json.RawMessage) {
	if !client.isCreator {
		h.sendError(ctx, client, "only the room host can kick users")
		return
	}
	var p KickPayload
	if err := json.Unmarshal(payload, &p); err != nil || p.UserID == "" {
		h.sendError(ctx, client, "invalid kick payload")
		return
	}
	if p.UserID == client.userID {
		h.sendError(ctx, client, "you cannot kick yourself")
		return
	}
	target := h.hub.FindClient(client.roomID, p.UserID)
	if target == nil {
		h.sendError(ctx, client, "user not found in room")
		return
	}
	h.hub.Broadcast(client.roomID, &message.Message{
		ID:        generateClientID(),
		RoomID:    client.roomID,
		Username:  target.username,
		Content:   target.username + " was kicked from the room",
		Type:      message.TypeSystem,
		Action:    message.ActionKick,
		CreatedAt: time.Now(),
	})
	h.hub.KickClient(target)
}

// handleBan bans a user from the room and kicks them if connected.
func (h *Handler) handleBan(ctx context.Context, client *Client, payload json.RawMessage) {
	if !client.isCreator {
		h.sendError(ctx, client, "only the room host can ban users")
		return
	}
	var p BanPayload
	if err := json.Unmarshal(payload, &p); err != nil || p.UserID == "" {
		h.sendError(ctx, client, "invalid ban payload")
		return
	}
	if p.UserID == client.userID {
		h.sendError(ctx, client, "you cannot ban yourself")
		return
	}
	target := h.hub.FindClient(client.roomID, p.UserID)
	targetName := p.UserID[:8]
	if target != nil {
		targetName = target.username
	}
	h.hub.Ban(client.roomID, p.UserID)
	h.hub.Broadcast(client.roomID, &message.Message{
		ID:        generateClientID(),
		RoomID:    client.roomID,
		Username:  targetName,
		Content:   targetName + " was banned from the room",
		Type:      message.TypeSystem,
		Action:    message.ActionBan,
		CreatedAt: time.Now(),
	})
	if target != nil {
		h.hub.KickClient(target)
	}
}

// handleMute toggles a user's mute status in the room.
func (h *Handler) handleMute(ctx context.Context, client *Client, payload json.RawMessage) {
	if !client.isCreator {
		h.sendError(ctx, client, "only the room host can mute users")
		return
	}
	var p MutePayload
	if err := json.Unmarshal(payload, &p); err != nil || p.UserID == "" {
		h.sendError(ctx, client, "invalid mute payload")
		return
	}
	if p.UserID == client.userID {
		h.sendError(ctx, client, "you cannot mute yourself")
		return
	}
	target := h.hub.FindClient(client.roomID, p.UserID)
	if target == nil {
		h.sendError(ctx, client, "user not found in room")
		return
	}
	muted := h.hub.Mute(client.roomID, p.UserID)
	var content string
	if muted {
		content = target.username + " was muted"
	} else {
		content = target.username + " was unmuted"
	}
	h.hub.Broadcast(client.roomID, &message.Message{
		ID:        generateClientID(),
		RoomID:    client.roomID,
		Username:  target.username,
		Content:   content,
		Type:      message.TypeSystem,
		Action:    message.ActionMute,
		CreatedAt: time.Now(),
	})
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
