package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/christopherjohns/chatsphere/internal/message"
	"nhooyr.io/websocket"
)

func newHandlerTestServer(t *testing.T, validateRoom RoomValidator) (*httptest.Server, *Hub, *SessionStore) {
	t.Helper()
	hub := NewHub(nil)
	sessions := NewSessionStore(30 * time.Second)
	messages := message.NewStore(200)
	hub.SetMessageStore(messages)
	hub.SetSessionStore(sessions)
	handler := NewHandler(hub, validateRoom, sessions, messages)
	return httptest.NewServer(handler), hub, sessions
}

func dialAndJoin(t *testing.T, url, roomID, username string) *websocket.Conn {
	t.Helper()
	conn, _ := dialJoinAndReadSession(t, url, roomID, username, "")

	// Drain the history envelope (always sent during join handshake).
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	if _, _, err := conn.Read(readCtx); err != nil {
		t.Fatalf("read history response error: %v", err)
	}

	return conn
}

func TestHandlerJoinAndChat(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Connect two clients to the same room.
	conn1 := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn1.Close(websocket.StatusNormalClosure, "")
	conn2 := dialAndJoin(t, ts.URL, "room1", "bob")
	defer conn2.Close(websocket.StatusNormalClosure, "")

	// Wait for both to register.
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.ClientCount("room1") != 2 {
		t.Fatalf("expected 2 clients, got %d", hub.ClientCount("room1"))
	}

	// conn1 receives the "alice joined" system message first, then "bob joined".
	// Drain system messages from conn1.
	drainSystemMessages(t, conn1, 2)

	// conn2 receives "bob joined".
	drainSystemMessages(t, conn2, 1)

	// alice sends a chat message.
	chatPayload, _ := json.Marshal(ChatPayload{Content: "hello everyone"})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn1.Write(ctx, websocket.MessageText, chatEnv); err != nil {
		t.Fatalf("write chat error: %v", err)
	}

	// Both conn1 and conn2 should receive the chat message.
	for _, conn := range []*websocket.Conn{conn1, conn2} {
		readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, data, err := conn.Read(readCtx)
		readCancel()
		if err != nil {
			t.Fatalf("read chat message error: %v", err)
		}

		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if env.Type != string(message.TypeChat) {
			t.Errorf("expected type 'chat', got %q", env.Type)
		}

		var msg message.Message
		if err := json.Unmarshal(env.Payload, &msg); err != nil {
			t.Fatalf("unmarshal payload error: %v", err)
		}
		if msg.Content != "hello everyone" {
			t.Errorf("expected content 'hello everyone', got %q", msg.Content)
		}
	}
}

func TestHandlerJoinInvalidRoom(t *testing.T) {
	ts, _, _ := newHandlerTestServer(t, func(roomID string) string {
		if roomID == "valid-room" {
			return ""
		}
		return "room not found"
	})
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send join with an invalid room.
	payload, _ := json.Marshal(JoinPayload{RoomID: "bad-room", Username: "alice"})
	env, _ := json.Marshal(Envelope{Type: "join", Payload: payload})

	writeCtx, writeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer writeCancel()
	if err := conn.Write(writeCtx, websocket.MessageText, env); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// The server should close the connection.
	readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer readCancel()
	_, _, err = conn.Read(readCtx)
	if err == nil {
		t.Fatal("expected connection to be closed for invalid room")
	}
}

func TestHandlerJoinMissingRoomID(t *testing.T) {
	ts, _, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Send join without room_id.
	payload, _ := json.Marshal(JoinPayload{Username: "alice"})
	env, _ := json.Marshal(Envelope{Type: "join", Payload: payload})

	writeCtx, writeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer writeCancel()
	if err := conn.Write(writeCtx, websocket.MessageText, env); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// The server should close the connection.
	readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer readCancel()
	_, _, err = conn.Read(readCtx)
	if err == nil {
		t.Fatal("expected connection to be closed for missing room_id")
	}
}

func TestHandlerDefaultUsername(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Join without a username.
	conn := dialAndJoin(t, ts.URL, "room1", "")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Read the system join message — the username should start with "anon-".
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)
	var msg message.Message
	json.Unmarshal(env.Payload, &msg)

	if !strings.HasPrefix(msg.Username, "anon-") {
		t.Errorf("expected username to start with 'anon-', got %q", msg.Username)
	}
}

// dialJoinAndReadSession connects, sends a join, and reads back the session envelope.
func dialJoinAndReadSession(t *testing.T, url, roomID, username, sessionID string) (*websocket.Conn, SessionPayload) {
	t.Helper()
	conn := dialWS(t, url)

	payload, _ := json.Marshal(JoinPayload{RoomID: roomID, Username: username, SessionID: sessionID})
	env, _ := json.Marshal(Envelope{Type: "join", Payload: payload})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, env); err != nil {
		t.Fatalf("write join error: %v", err)
	}

	// Read the session response.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read session response error: %v", err)
	}

	var sessEnv Envelope
	if err := json.Unmarshal(data, &sessEnv); err != nil {
		t.Fatalf("unmarshal session envelope error: %v", err)
	}
	if sessEnv.Type != "session" {
		t.Fatalf("expected type 'session', got %q", sessEnv.Type)
	}

	var sp SessionPayload
	if err := json.Unmarshal(sessEnv.Payload, &sp); err != nil {
		t.Fatalf("unmarshal session payload error: %v", err)
	}
	return conn, sp
}

func TestHandlerSessionResumption(t *testing.T) {
	ts, hub, sessions := newHandlerTestServer(t, nil)
	defer ts.Close()

	// 1. Connect and get a session ID.
	conn1, sp1 := dialJoinAndReadSession(t, ts.URL, "room1", "alice", "")
	if sp1.Resumed {
		t.Fatal("first connection should not be resumed")
	}
	if sp1.SessionID == "" {
		t.Fatal("expected a session ID")
	}
	if sp1.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", sp1.Username)
	}

	// Wait for client to be registered.
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Drain history + "alice joined" system message.
	drainSystemMessages(t, conn1, 2)

	// 2. Disconnect the first client.
	conn1.Close(websocket.StatusNormalClosure, "")

	// Wait for client to be removed.
	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Session should be marked disconnected but still exist.
	if sessions.Count() == 0 {
		t.Fatal("session should still exist after disconnect")
	}

	// 3. Reconnect with the same session ID — should resume.
	conn2, sp2 := dialJoinAndReadSession(t, ts.URL, "room1", "", sp1.SessionID)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	if !sp2.Resumed {
		t.Fatal("reconnection should have resumed the session")
	}
	if sp2.SessionID != sp1.SessionID {
		t.Errorf("expected same session ID %q, got %q", sp1.SessionID, sp2.SessionID)
	}
	if sp2.UserID != sp1.UserID {
		t.Errorf("expected same user ID %q, got %q", sp1.UserID, sp2.UserID)
	}
	if sp2.Username != "alice" {
		t.Errorf("expected username 'alice' preserved, got %q", sp2.Username)
	}

	// No backfill expected here because alice was the only client and no
	// messages were sent while she was disconnected. The "alice left" system
	// message was sent while she was still in the hub, updating her LastMessageID.

	// Wait for client to be registered.
	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// No "joined" system message should have been broadcast for a resumed session.
	// Verify by sending a chat and confirming the next message is the chat, not a system message.
	chatPayload, _ := json.Marshal(ChatPayload{Content: "I'm back"})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn2.Write(ctx, websocket.MessageText, chatEnv); err != nil {
		t.Fatalf("write chat error: %v", err)
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn2.Read(readCtx)
	if err != nil {
		t.Fatalf("read chat error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if env.Type != string(message.TypeChat) {
		t.Errorf("expected first message after resume to be 'chat', got %q (should not get 'joined' system message)", env.Type)
	}
}

func TestHandlerSessionResumptionWrongRoom(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Connect to room1.
	conn1, sp1 := dialJoinAndReadSession(t, ts.URL, "room1", "alice", "")
	drainSystemMessages(t, conn1, 2) // history + "alice joined"

	// Disconnect.
	conn1.Close(websocket.StatusNormalClosure, "")
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Try to resume in room2 — should create a new session (not resumed).
	conn2, sp2 := dialJoinAndReadSession(t, ts.URL, "room2", "alice", sp1.SessionID)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	if sp2.Resumed {
		t.Fatal("should not resume session in a different room")
	}
	if sp2.SessionID == sp1.SessionID {
		t.Error("should have created a new session")
	}
}

func TestHandlerSessionResumptionExpired(t *testing.T) {
	hub := NewHub(nil)
	sessions := NewSessionStore(50 * time.Millisecond) // Very short TTL for testing.
	messages := message.NewStore(200)
	hub.SetMessageStore(messages)
	hub.SetSessionStore(sessions)
	handler := NewHandler(hub, nil, sessions, messages)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Connect and get session.
	conn1, sp1 := dialJoinAndReadSession(t, ts.URL, "room1", "alice", "")
	drainSystemMessages(t, conn1, 2) // history + "alice joined"

	// Disconnect.
	conn1.Close(websocket.StatusNormalClosure, "")
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for session to expire (TTL=50ms, reap runs every 25ms).
	time.Sleep(150 * time.Millisecond)

	// Try to resume — should fail because session expired (creates new session).
	conn2, sp2 := dialJoinAndReadSession(t, ts.URL, "room1", "alice", sp1.SessionID)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	if sp2.Resumed {
		t.Fatal("should not resume an expired session")
	}
	if sp2.SessionID == sp1.SessionID {
		t.Error("should have created a new session for expired session")
	}
}

func TestHandlerBackfillOnReconnect(t *testing.T) {
	hub := NewHub(nil)
	sessions := NewSessionStore(30 * time.Second)
	messages := message.NewStore(200)
	hub.SetMessageStore(messages)
	hub.SetSessionStore(sessions)
	handler := NewHandler(hub, nil, sessions, messages)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 1. Alice and Bob connect to room1.
	conn1, sp1 := dialJoinAndReadSession(t, ts.URL, "room1", "alice", "")
	conn2 := dialAndJoin(t, ts.URL, "room1", "bob")
	defer conn2.Close(websocket.StatusNormalClosure, "")

	// Wait for both to register.
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Drain history + system messages from conn1 (history, alice joined, bob joined).
	drainSystemMessages(t, conn1, 3)
	drainSystemMessages(t, conn2, 1) // "bob joined"

	// 2. Alice disconnects.
	conn1.Close(websocket.StatusNormalClosure, "")
	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Drain "alice left" from Bob's connection.
	drainSystemMessages(t, conn2, 1)

	// 3. Bob sends some messages while Alice is offline.
	for _, content := range []string{"msg1", "msg2", "msg3"} {
		chatPayload, _ := json.Marshal(ChatPayload{Content: content})
		chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := conn2.Write(ctx, websocket.MessageText, chatEnv); err != nil {
			cancel()
			t.Fatalf("write chat error: %v", err)
		}
		cancel()

		// Bob receives his own message.
		readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _, err := conn2.Read(readCtx)
		readCancel()
		if err != nil {
			t.Fatalf("bob read own message error: %v", err)
		}
	}

	// 4. Alice reconnects — should receive backfill.
	conn3, sp3 := dialJoinAndReadSession(t, ts.URL, "room1", "", sp1.SessionID)
	defer conn3.Close(websocket.StatusNormalClosure, "")

	if !sp3.Resumed {
		t.Fatal("expected session to be resumed")
	}

	// Read the backfill envelope.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, backfillData, err := conn3.Read(readCtx)
	if err != nil {
		t.Fatalf("read backfill error: %v", err)
	}

	var backfillEnv Envelope
	if err := json.Unmarshal(backfillData, &backfillEnv); err != nil {
		t.Fatalf("unmarshal backfill envelope error: %v", err)
	}
	if backfillEnv.Type != "backfill" {
		t.Fatalf("expected type 'backfill', got %q", backfillEnv.Type)
	}

	var backfillPayload BackfillPayload
	if err := json.Unmarshal(backfillEnv.Payload, &backfillPayload); err != nil {
		t.Fatalf("unmarshal backfill payload error: %v", err)
	}

	if backfillPayload.HasGap {
		t.Error("expected has_gap to be false for normal backfill")
	}

	// Should contain: "msg1" + "msg2" + "msg3" (chat messages sent while alice was offline).
	// Note: "alice left" is NOT included because it was broadcast while alice was still
	// in the hub, which updated her LastMessageID past it.
	if len(backfillPayload.Messages) != 3 {
		t.Fatalf("expected 3 backfill messages, got %d", len(backfillPayload.Messages))
	}

	for i, content := range []string{"msg1", "msg2", "msg3"} {
		if backfillPayload.Messages[i].Content != content {
			t.Errorf("backfill[%d]: expected content %q, got %q", i, content, backfillPayload.Messages[i].Content)
		}
		if backfillPayload.Messages[i].Type != message.TypeChat {
			t.Errorf("backfill[%d]: expected type 'chat', got %q", i, backfillPayload.Messages[i].Type)
		}
	}
}

func TestHandlerHistoryOnNewConnection(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Connect and immediately send a chat to populate the message store.
	conn1 := dialAndJoin(t, ts.URL, "room1", "alice")
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn1, 1) // "alice joined"

	chatPayload, _ := json.Marshal(ChatPayload{Content: "hello"})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn1.Write(ctx, websocket.MessageText, chatEnv)
	drainSystemMessages(t, conn1, 1) // the chat message

	// New client joins — should receive history with existing messages.
	// Use dialJoinAndReadSession so we can inspect the history envelope directly.
	conn2, _ := dialJoinAndReadSession(t, ts.URL, "room1", "bob", "")
	defer conn2.Close(websocket.StatusNormalClosure, "")
	defer conn1.Close(websocket.StatusNormalClosure, "")

	// Read the history envelope (sent right after session during join).
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn2.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if env.Type != "history" {
		t.Fatalf("expected type 'history', got %q", env.Type)
	}

	var historyMsgs []message.Message
	if err := json.Unmarshal(env.Payload, &historyMsgs); err != nil {
		t.Fatalf("unmarshal history messages error: %v", err)
	}

	// Should contain "alice joined" system message and "hello" chat message.
	if len(historyMsgs) < 1 {
		t.Fatal("expected at least 1 history message")
	}

	// The last message should be the "hello" chat.
	last := historyMsgs[len(historyMsgs)-1]
	if last.Content != "hello" {
		t.Errorf("expected last history message to be 'hello', got %q", last.Content)
	}
	if last.Type != message.TypeChat {
		t.Errorf("expected type 'chat', got %q", last.Type)
	}
}

func TestHandlerJoinRoomFull(t *testing.T) {
	full := false
	ts, hub, _ := newHandlerTestServer(t, func(roomID string) string {
		if full {
			return "room is full"
		}
		return ""
	})
	defer ts.Close()

	// First client joins successfully.
	conn1 := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn1.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Mark room as full.
	full = true

	// Second client should be rejected.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn2, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")

	payload, _ := json.Marshal(JoinPayload{RoomID: "room1", Username: "bob"})
	env, _ := json.Marshal(Envelope{Type: "join", Payload: payload})

	writeCtx, writeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer writeCancel()
	if err := conn2.Write(writeCtx, websocket.MessageText, env); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// The server should close the connection because the room is full.
	readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer readCancel()
	_, _, err = conn2.Read(readCtx)
	if err == nil {
		t.Fatal("expected connection to be closed for full room")
	}

	// Only the first client should remain.
	if hub.ClientCount("room1") != 1 {
		t.Errorf("expected 1 client, got %d", hub.ClientCount("room1"))
	}
}

func TestHandlerLeaveUpdatesClientCount(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Two clients join.
	conn1 := dialAndJoin(t, ts.URL, "room1", "alice")
	conn2 := dialAndJoin(t, ts.URL, "room1", "bob")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.ClientCount("room1") != 2 {
		t.Fatalf("expected 2 clients, got %d", hub.ClientCount("room1"))
	}

	// Drain system messages from both.
	drainSystemMessages(t, conn1, 2) // "alice joined" + "bob joined"
	drainSystemMessages(t, conn2, 1) // "bob joined"

	// Alice disconnects.
	conn1.Close(websocket.StatusNormalClosure, "")

	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.ClientCount("room1") != 1 {
		t.Fatalf("expected 1 client after alice left, got %d", hub.ClientCount("room1"))
	}

	// Bob should receive "alice left" system message.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn2.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)
	var msg message.Message
	json.Unmarshal(env.Payload, &msg)

	if msg.Type != message.TypeSystem {
		t.Errorf("expected system message, got %q", msg.Type)
	}
	if !strings.Contains(msg.Content, "left") {
		t.Errorf("expected 'left' in message content, got %q", msg.Content)
	}
	if msg.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", msg.Username)
	}

	// Bob disconnects — room should be empty.
	conn2.Close(websocket.StatusNormalClosure, "")

	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.ClientCount("room1") != 0 {
		t.Fatalf("expected 0 clients after all left, got %d", hub.ClientCount("room1"))
	}
}

func TestHandlerJoinBroadcastsSystemMessage(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Alice joins first.
	conn1 := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn1.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Drain "alice joined" from conn1.
	drainSystemMessages(t, conn1, 1)

	// Bob joins — alice should receive "bob joined" system message.
	conn2 := dialAndJoin(t, ts.URL, "room1", "bob")
	defer conn2.Close(websocket.StatusNormalClosure, "")

	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn1.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)
	var msg message.Message
	json.Unmarshal(env.Payload, &msg)

	if msg.Type != message.TypeSystem {
		t.Errorf("expected system message, got %q", msg.Type)
	}
	if !strings.Contains(msg.Content, "joined") {
		t.Errorf("expected 'joined' in content, got %q", msg.Content)
	}
	if msg.Username != "bob" {
		t.Errorf("expected username 'bob', got %q", msg.Username)
	}
}

func TestHandlerChatEmptyContent(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	conn := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn, 1) // "alice joined"

	// Send empty content.
	chatPayload, _ := json.Marshal(ChatPayload{Content: ""})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, chatEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Should receive an error envelope, not a chat broadcast.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if env.Type != "error" {
		t.Errorf("expected type 'error', got %q", env.Type)
	}

	var errPayload ErrorPayload
	if err := json.Unmarshal(env.Payload, &errPayload); err != nil {
		t.Fatalf("unmarshal error payload: %v", err)
	}
	if errPayload.Message != "message content is required" {
		t.Errorf("expected 'message content is required', got %q", errPayload.Message)
	}
}

func TestHandlerChatWhitespaceOnly(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	conn := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn, 1)

	// Send whitespace-only content.
	chatPayload, _ := json.Marshal(ChatPayload{Content: "   \t\n  "})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, chatEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Should receive an error, not a broadcast.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)
	if env.Type != "error" {
		t.Errorf("expected type 'error', got %q", env.Type)
	}
}

func TestHandlerChatContentTrimmed(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	conn := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn, 1)

	// Send content with leading/trailing whitespace.
	chatPayload, _ := json.Marshal(ChatPayload{Content: "  hello world  "})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, chatEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)
	if env.Type != "chat" {
		t.Fatalf("expected type 'chat', got %q", env.Type)
	}

	var msg message.Message
	json.Unmarshal(env.Payload, &msg)
	if msg.Content != "hello world" {
		t.Errorf("expected trimmed content 'hello world', got %q", msg.Content)
	}
}

func TestHandlerChatExceedsMaxLength(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	conn := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn, 1)

	// Send content that exceeds the max length.
	longContent := strings.Repeat("a", maxMessageLength+1)
	chatPayload, _ := json.Marshal(ChatPayload{Content: longContent})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, chatEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)
	if env.Type != "error" {
		t.Errorf("expected type 'error', got %q", env.Type)
	}

	var errPayload ErrorPayload
	json.Unmarshal(env.Payload, &errPayload)
	if !strings.Contains(errPayload.Message, "maximum length") {
		t.Errorf("expected max length error, got %q", errPayload.Message)
	}
}

func TestHandlerChatAtMaxLength(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	conn := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn, 1)

	// Send content exactly at max length — should succeed.
	exactContent := strings.Repeat("b", maxMessageLength)
	chatPayload, _ := json.Marshal(ChatPayload{Content: exactContent})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, chatEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)
	if env.Type != "chat" {
		t.Errorf("expected type 'chat' for max-length message, got %q", env.Type)
	}
}

func TestHandlerChatBroadcastToMultipleClients(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Connect three clients to the same room.
	conn1 := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn1.Close(websocket.StatusNormalClosure, "")
	conn2 := dialAndJoin(t, ts.URL, "room1", "bob")
	defer conn2.Close(websocket.StatusNormalClosure, "")
	conn3 := dialAndJoin(t, ts.URL, "room1", "charlie")
	defer conn3.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") < 3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.ClientCount("room1") != 3 {
		t.Fatalf("expected 3 clients, got %d", hub.ClientCount("room1"))
	}

	// Drain system messages.
	drainSystemMessages(t, conn1, 3) // alice joined, bob joined, charlie joined
	drainSystemMessages(t, conn2, 2) // bob joined, charlie joined
	drainSystemMessages(t, conn3, 1) // charlie joined

	// Alice sends a message.
	chatPayload, _ := json.Marshal(ChatPayload{Content: "hello from alice"})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn1.Write(ctx, websocket.MessageText, chatEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// All three should receive the message.
	for i, conn := range []*websocket.Conn{conn1, conn2, conn3} {
		readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, data, err := conn.Read(readCtx)
		readCancel()
		if err != nil {
			t.Fatalf("client %d read error: %v", i, err)
		}

		var env Envelope
		json.Unmarshal(data, &env)
		if env.Type != "chat" {
			t.Errorf("client %d: expected 'chat', got %q", i, env.Type)
		}

		var msg message.Message
		json.Unmarshal(env.Payload, &msg)
		if msg.Content != "hello from alice" {
			t.Errorf("client %d: expected 'hello from alice', got %q", i, msg.Content)
		}
		if msg.Username != "alice" {
			t.Errorf("client %d: expected username 'alice', got %q", i, msg.Username)
		}
	}
}

func TestHandlerChatRoomIsolation(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Two clients in different rooms.
	conn1 := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn1.Close(websocket.StatusNormalClosure, "")
	conn2 := dialAndJoin(t, ts.URL, "room2", "bob")
	defer conn2.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for (hub.ClientCount("room1") == 0 || hub.ClientCount("room2") == 0) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	drainSystemMessages(t, conn1, 1) // alice joined
	drainSystemMessages(t, conn2, 1) // bob joined

	// Alice sends a message in room1.
	chatPayload, _ := json.Marshal(ChatPayload{Content: "room1 only"})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn1.Write(ctx, websocket.MessageText, chatEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// conn1 should receive the message.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, data, err := conn1.Read(readCtx)
	readCancel()
	if err != nil {
		t.Fatalf("conn1 read error: %v", err)
	}
	var env Envelope
	json.Unmarshal(data, &env)
	if env.Type != "chat" {
		t.Errorf("expected 'chat', got %q", env.Type)
	}

	// conn2 should NOT receive the message.
	readCtx2, readCancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer readCancel2()
	_, _, err = conn2.Read(readCtx2)
	if err == nil {
		t.Fatal("conn2 should not receive messages from room1")
	}
}

func TestHandlerMessagePersistence(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	conn := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn, 1)

	// Send a message.
	chatPayload, _ := json.Marshal(ChatPayload{Content: "persistent msg"})
	chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, chatEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Read back the broadcast.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)
	var msg message.Message
	json.Unmarshal(env.Payload, &msg)

	// Verify message fields.
	if msg.ID == "" {
		t.Error("expected message to have an ID")
	}
	if msg.RoomID != "room1" {
		t.Errorf("expected room_id 'room1', got %q", msg.RoomID)
	}
	if msg.UserID == "" {
		t.Error("expected message to have a user_id")
	}
	if msg.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", msg.Username)
	}
	if msg.Content != "persistent msg" {
		t.Errorf("expected content 'persistent msg', got %q", msg.Content)
	}
	if msg.Type != message.TypeChat {
		t.Errorf("expected type 'chat', got %q", msg.Type)
	}
	if msg.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestHandlerHistoryEmptyRoom(t *testing.T) {
	ts, _, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Join a room with no prior messages.
	conn, _ := dialJoinAndReadSession(t, ts.URL, "room1", "alice", "")
	defer conn.Close(websocket.StatusNormalClosure, "")

	// History envelope is always sent; for an empty room it should be an empty array.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if env.Type != "history" {
		t.Fatalf("expected type 'history', got %q", env.Type)
	}

	var historyMsgs []message.Message
	if err := json.Unmarshal(env.Payload, &historyMsgs); err != nil {
		t.Fatalf("unmarshal history error: %v", err)
	}
	if len(historyMsgs) != 0 {
		t.Errorf("expected 0 history messages for empty room, got %d", len(historyMsgs))
	}
}

func TestHandlerHistoryLimit(t *testing.T) {
	hub := NewHub(nil)
	sessions := NewSessionStore(30 * time.Second)
	messages := message.NewStore(200)
	hub.SetMessageStore(messages)
	hub.SetSessionStore(sessions)
	handler := NewHandler(hub, nil, sessions, messages)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Pre-populate the store with 60 messages.
	for i := 0; i < 60; i++ {
		messages.Append(&message.Message{
			ID:        fmt.Sprintf("msg-%d", i),
			RoomID:    "room1",
			Content:   fmt.Sprintf("message %d", i),
			Type:      message.TypeChat,
			CreatedAt: time.Now(),
		})
	}

	// Join the room and read the history envelope.
	conn, _ := dialJoinAndReadSession(t, ts.URL, "room1", "alice", "")
	defer conn.Close(websocket.StatusNormalClosure, "")

	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if env.Type != "history" {
		t.Fatalf("expected type 'history', got %q", env.Type)
	}

	var historyMsgs []message.Message
	if err := json.Unmarshal(env.Payload, &historyMsgs); err != nil {
		t.Fatalf("unmarshal history error: %v", err)
	}

	// Should receive exactly 50 messages (the limit), not all 60.
	if len(historyMsgs) != 50 {
		t.Fatalf("expected 50 history messages, got %d", len(historyMsgs))
	}

	// Should be the last 50 messages (IDs msg-10 through msg-59).
	if historyMsgs[0].ID != "msg-10" {
		t.Errorf("expected first history message ID 'msg-10', got %q", historyMsgs[0].ID)
	}
	if historyMsgs[49].ID != "msg-59" {
		t.Errorf("expected last history message ID 'msg-59', got %q", historyMsgs[49].ID)
	}
}

func TestHandlerBackfillGapOnEvictedMessage(t *testing.T) {
	hub := NewHub(nil)
	sessions := NewSessionStore(30 * time.Second)
	// Small store: only holds 5 messages per room.
	messages := message.NewStore(5)
	hub.SetMessageStore(messages)
	hub.SetSessionStore(sessions)
	handler := NewHandler(hub, nil, sessions, messages)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 1. Alice and Bob connect.
	conn1, sp1 := dialJoinAndReadSession(t, ts.URL, "room1", "alice", "")
	conn2 := dialAndJoin(t, ts.URL, "room1", "bob")
	defer conn2.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Drain system messages.
	drainSystemMessages(t, conn1, 3) // history, alice joined, bob joined
	drainSystemMessages(t, conn2, 1) // bob joined

	// 2. Alice disconnects.
	conn1.Close(websocket.StatusNormalClosure, "")
	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn2, 1) // "alice left"

	// 3. Bob sends 10 messages — more than the store can hold (5).
	// This will evict Alice's LastMessageID from the ring buffer.
	for i := 0; i < 10; i++ {
		chatPayload, _ := json.Marshal(ChatPayload{Content: fmt.Sprintf("overflow-%d", i)})
		chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := conn2.Write(ctx, websocket.MessageText, chatEnv); err != nil {
			cancel()
			t.Fatalf("write error: %v", err)
		}
		cancel()
		readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _, err := conn2.Read(readCtx)
		readCancel()
		if err != nil {
			t.Fatalf("bob read own message error: %v", err)
		}
	}

	// 4. Alice reconnects — should receive backfill with has_gap=true.
	conn3, sp3 := dialJoinAndReadSession(t, ts.URL, "room1", "", sp1.SessionID)
	defer conn3.Close(websocket.StatusNormalClosure, "")

	if !sp3.Resumed {
		t.Fatal("expected session to be resumed")
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, backfillData, err := conn3.Read(readCtx)
	if err != nil {
		t.Fatalf("read backfill error: %v", err)
	}

	var backfillEnv Envelope
	if err := json.Unmarshal(backfillData, &backfillEnv); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if backfillEnv.Type != "backfill" {
		t.Fatalf("expected type 'backfill', got %q", backfillEnv.Type)
	}

	var backfillPayload BackfillPayload
	if err := json.Unmarshal(backfillEnv.Payload, &backfillPayload); err != nil {
		t.Fatalf("unmarshal backfill payload error: %v", err)
	}

	if !backfillPayload.HasGap {
		t.Error("expected has_gap=true when LastMessageID was evicted from store")
	}

	// Should receive the most recent messages (capped by store size).
	if len(backfillPayload.Messages) == 0 {
		t.Fatal("expected some backfill messages")
	}
	if len(backfillPayload.Messages) > 5 {
		t.Errorf("expected at most 5 messages (store size), got %d", len(backfillPayload.Messages))
	}

	// Last message should be the most recent one sent.
	last := backfillPayload.Messages[len(backfillPayload.Messages)-1]
	if last.Content != "overflow-9" {
		t.Errorf("expected last backfill message to be 'overflow-9', got %q", last.Content)
	}
}

func TestHandlerBackfillNoGapOnNormalReconnect(t *testing.T) {
	hub := NewHub(nil)
	sessions := NewSessionStore(30 * time.Second)
	messages := message.NewStore(200)
	hub.SetMessageStore(messages)
	hub.SetSessionStore(sessions)
	handler := NewHandler(hub, nil, sessions, messages)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 1. Alice and Bob connect.
	conn1, sp1 := dialJoinAndReadSession(t, ts.URL, "room1", "alice", "")
	conn2 := dialAndJoin(t, ts.URL, "room1", "bob")
	defer conn2.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	drainSystemMessages(t, conn1, 3)
	drainSystemMessages(t, conn2, 1)

	// 2. Alice disconnects.
	conn1.Close(websocket.StatusNormalClosure, "")
	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn2, 1)

	// 3. Bob sends 2 messages (well within store capacity).
	for _, content := range []string{"hello", "world"} {
		chatPayload, _ := json.Marshal(ChatPayload{Content: content})
		chatEnv, _ := json.Marshal(Envelope{Type: "chat", Payload: chatPayload})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := conn2.Write(ctx, websocket.MessageText, chatEnv); err != nil {
			cancel()
			t.Fatalf("write error: %v", err)
		}
		cancel()
		readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _, err := conn2.Read(readCtx)
		readCancel()
		if err != nil {
			t.Fatalf("bob read error: %v", err)
		}
	}

	// 4. Alice reconnects.
	conn3, sp3 := dialJoinAndReadSession(t, ts.URL, "room1", "", sp1.SessionID)
	defer conn3.Close(websocket.StatusNormalClosure, "")

	if !sp3.Resumed {
		t.Fatal("expected session to be resumed")
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, backfillData, err := conn3.Read(readCtx)
	if err != nil {
		t.Fatalf("read backfill error: %v", err)
	}

	var backfillEnv Envelope
	json.Unmarshal(backfillData, &backfillEnv)

	var backfillPayload BackfillPayload
	json.Unmarshal(backfillEnv.Payload, &backfillPayload)

	if backfillPayload.HasGap {
		t.Error("expected has_gap=false for normal backfill within store capacity")
	}
	if len(backfillPayload.Messages) != 2 {
		t.Errorf("expected 2 backfill messages, got %d", len(backfillPayload.Messages))
	}
}

// drainSystemMessages reads and discards n messages from the connection.
func drainSystemMessages(t *testing.T, conn *websocket.Conn, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _, err := conn.Read(ctx)
		cancel()
		if err != nil {
			t.Fatalf("drain message %d: %v", i, err)
		}
	}
}

func TestHandlerLeaveMessage(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Two clients join the same room.
	conn1 := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn1.Close(websocket.StatusNormalClosure, "")
	conn2 := dialAndJoin(t, ts.URL, "room1", "bob")
	defer conn2.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.ClientCount("room1") != 2 {
		t.Fatalf("expected 2 clients, got %d", hub.ClientCount("room1"))
	}

	// Drain system messages.
	drainSystemMessages(t, conn1, 2) // "alice joined", "bob joined"
	drainSystemMessages(t, conn2, 1) // "bob joined"

	// Alice sends an explicit leave message.
	leaveEnv, _ := json.Marshal(Envelope{Type: "leave", Payload: json.RawMessage(`{}`)})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn1.Write(ctx, websocket.MessageText, leaveEnv); err != nil {
		t.Fatalf("write leave error: %v", err)
	}

	// Bob should receive "alice left" system message.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn2.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)
	var msg message.Message
	json.Unmarshal(env.Payload, &msg)

	if msg.Type != message.TypeSystem {
		t.Errorf("expected system message, got %q", msg.Type)
	}
	if !strings.Contains(msg.Content, "left") {
		t.Errorf("expected 'left' in content, got %q", msg.Content)
	}
	if msg.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", msg.Username)
	}

	// Client count should decrease.
	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.ClientCount("room1") != 1 {
		t.Fatalf("expected 1 client after leave, got %d", hub.ClientCount("room1"))
	}
}

func TestHandlerLeaveAllowsSessionResumption(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	// Alice joins and gets a session.
	conn1, sp1 := dialJoinAndReadSession(t, ts.URL, "room1", "alice", "")
	drainSystemMessages(t, conn1, 2) // history + "alice joined"

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Alice sends explicit leave.
	leaveEnv, _ := json.Marshal(Envelope{Type: "leave", Payload: json.RawMessage(`{}`)})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn1.Write(ctx, websocket.MessageText, leaveEnv); err != nil {
		t.Fatalf("write leave error: %v", err)
	}

	// Wait for cleanup.
	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Alice should be able to resume the session.
	conn2, sp2 := dialJoinAndReadSession(t, ts.URL, "room1", "", sp1.SessionID)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	if !sp2.Resumed {
		t.Fatal("expected session to be resumed after explicit leave")
	}
	if sp2.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", sp2.Username)
	}
}

func TestHandlerJoinUsernameTooLong(t *testing.T) {
	ts, _, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	longUsername := strings.Repeat("a", maxUsernameLength+1)
	payload, _ := json.Marshal(JoinPayload{RoomID: "room1", Username: longUsername})
	env, _ := json.Marshal(Envelope{Type: "join", Payload: payload})

	writeCtx, writeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer writeCancel()
	if err := conn.Write(writeCtx, websocket.MessageText, env); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Connection should be closed with policy violation.
	readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer readCancel()
	_, _, err = conn.Read(readCtx)
	if err == nil {
		t.Fatal("expected connection to be closed for too-long username")
	}
}

func TestHandlerJoinUsernameAtMaxLength(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	exactUsername := strings.Repeat("a", maxUsernameLength)
	conn := dialAndJoin(t, ts.URL, "room1", exactUsername)
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.ClientCount("room1") != 1 {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount("room1"))
	}
}

func TestHandlerHistoryFetch(t *testing.T) {
	hub := NewHub(nil)
	sessions := NewSessionStore(30 * time.Second)
	messages := message.NewStore(200)
	hub.SetMessageStore(messages)
	hub.SetSessionStore(sessions)
	handler := NewHandler(hub, nil, sessions, messages)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Pre-populate the store with 80 messages.
	for i := 0; i < 80; i++ {
		messages.Append(&message.Message{
			ID:        fmt.Sprintf("msg-%d", i),
			RoomID:    "room1",
			Content:   fmt.Sprintf("message %d", i),
			Type:      message.TypeChat,
			CreatedAt: time.Now(),
		})
	}

	// Join the room — receives last 50 messages (msg-30 to msg-79).
	conn := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn, 1) // "alice joined"

	// Request older messages before msg-30 (the oldest message in history).
	fetchPayload, _ := json.Marshal(HistoryFetchPayload{BeforeID: "msg-30", Limit: 20})
	fetchEnv, _ := json.Marshal(Envelope{Type: "history_fetch", Payload: fetchPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, fetchEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// Read the history_batch response.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if env.Type != "history_batch" {
		t.Fatalf("expected type 'history_batch', got %q", env.Type)
	}

	var batch HistoryBatchPayload
	if err := json.Unmarshal(env.Payload, &batch); err != nil {
		t.Fatalf("unmarshal batch error: %v", err)
	}

	if len(batch.Messages) != 20 {
		t.Fatalf("expected 20 messages, got %d", len(batch.Messages))
	}
	// Should be messages 10-29.
	if batch.Messages[0].ID != "msg-10" {
		t.Errorf("expected first message 'msg-10', got %q", batch.Messages[0].ID)
	}
	if batch.Messages[19].ID != "msg-29" {
		t.Errorf("expected last message 'msg-29', got %q", batch.Messages[19].ID)
	}
	if !batch.HasMore {
		t.Error("expected has_more=true since there are more messages before msg-10")
	}
}

func TestHandlerHistoryFetchNoMore(t *testing.T) {
	hub := NewHub(nil)
	sessions := NewSessionStore(30 * time.Second)
	messages := message.NewStore(200)
	hub.SetMessageStore(messages)
	hub.SetSessionStore(sessions)
	handler := NewHandler(hub, nil, sessions, messages)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Pre-populate with just 5 messages.
	for i := 0; i < 5; i++ {
		messages.Append(&message.Message{
			ID:        fmt.Sprintf("msg-%d", i),
			RoomID:    "room1",
			Content:   fmt.Sprintf("message %d", i),
			Type:      message.TypeChat,
			CreatedAt: time.Now(),
		})
	}

	conn := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn, 1) // "alice joined"

	// Request messages before msg-2 with limit 50.
	fetchPayload, _ := json.Marshal(HistoryFetchPayload{BeforeID: "msg-2", Limit: 50})
	fetchEnv, _ := json.Marshal(Envelope{Type: "history_fetch", Payload: fetchPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, fetchEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)
	if env.Type != "history_batch" {
		t.Fatalf("expected type 'history_batch', got %q", env.Type)
	}

	var batch HistoryBatchPayload
	json.Unmarshal(env.Payload, &batch)

	// Only 2 messages before msg-2 (msg-0, msg-1).
	if len(batch.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(batch.Messages))
	}
	if batch.Messages[0].ID != "msg-0" {
		t.Errorf("expected first message 'msg-0', got %q", batch.Messages[0].ID)
	}
	if batch.HasMore {
		t.Error("expected has_more=false since these are the oldest messages")
	}
}

func TestHandlerHistoryFetchLimitCapped(t *testing.T) {
	hub := NewHub(nil)
	sessions := NewSessionStore(30 * time.Second)
	messages := message.NewStore(200)
	hub.SetMessageStore(messages)
	hub.SetSessionStore(sessions)
	handler := NewHandler(hub, nil, sessions, messages)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Pre-populate with 150 messages.
	for i := 0; i < 150; i++ {
		messages.Append(&message.Message{
			ID:        fmt.Sprintf("msg-%d", i),
			RoomID:    "room1",
			Content:   fmt.Sprintf("message %d", i),
			Type:      message.TypeChat,
			CreatedAt: time.Now(),
		})
	}

	conn := dialAndJoin(t, ts.URL, "room1", "alice")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	drainSystemMessages(t, conn, 1)

	// Request with limit exceeding max (100).
	fetchPayload, _ := json.Marshal(HistoryFetchPayload{BeforeID: "msg-149", Limit: 500})
	fetchEnv, _ := json.Marshal(Envelope{Type: "history_fetch", Payload: fetchPayload})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, fetchEnv); err != nil {
		t.Fatalf("write error: %v", err)
	}

	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	_, data, err := conn.Read(readCtx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	json.Unmarshal(data, &env)

	var batch HistoryBatchPayload
	json.Unmarshal(env.Payload, &batch)

	// Should be capped at 100.
	if len(batch.Messages) != 100 {
		t.Fatalf("expected 100 messages (capped), got %d", len(batch.Messages))
	}
	if !batch.HasMore {
		t.Error("expected has_more=true")
	}
}

func TestHandlerJoinUsernameTrimmed(t *testing.T) {
	ts, hub, _ := newHandlerTestServer(t, nil)
	defer ts.Close()

	conn, sp := dialJoinAndReadSession(t, ts.URL, "room1", "  alice  ", "")
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if sp.Username != "alice" {
		t.Errorf("expected trimmed username 'alice', got %q", sp.Username)
	}
}
