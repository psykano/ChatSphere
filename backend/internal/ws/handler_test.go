package ws

import (
	"context"
	"encoding/json"
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
	conn := dialWS(t, url)

	payload, _ := json.Marshal(JoinPayload{RoomID: roomID, Username: username})
	env, _ := json.Marshal(Envelope{Type: "join", Payload: payload})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, env); err != nil {
		t.Fatalf("write join error: %v", err)
	}

	// Drain the session response envelope.
	readCtx, readCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer readCancel()
	if _, _, err := conn.Read(readCtx); err != nil {
		t.Fatalf("read session response error: %v", err)
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

	// Drain the "alice joined" system message.
	drainSystemMessages(t, conn1, 1)

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
	drainSystemMessages(t, conn1, 1) // "alice joined"

	// Disconnect.
	conn1.Close(websocket.StatusNormalClosure, "")
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Try to resume in room2 — should create a new session.
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
	drainSystemMessages(t, conn1, 1)

	// Disconnect.
	conn1.Close(websocket.StatusNormalClosure, "")
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for session to expire (TTL=50ms, reap runs every 25ms).
	time.Sleep(150 * time.Millisecond)

	// Try to resume — should fail because session expired.
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

	// Drain system messages (alice joined, bob joined) from both connections.
	drainSystemMessages(t, conn1, 2) // "alice joined" + "bob joined"
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

	var backfillMsgs []message.Message
	if err := json.Unmarshal(backfillEnv.Payload, &backfillMsgs); err != nil {
		t.Fatalf("unmarshal backfill messages error: %v", err)
	}

	// Should contain: "msg1" + "msg2" + "msg3" (chat messages sent while alice was offline).
	// Note: "alice left" is NOT included because it was broadcast while alice was still
	// in the hub, which updated her LastMessageID past it.
	if len(backfillMsgs) != 3 {
		t.Fatalf("expected 3 backfill messages, got %d", len(backfillMsgs))
	}

	for i, content := range []string{"msg1", "msg2", "msg3"} {
		if backfillMsgs[i].Content != content {
			t.Errorf("backfill[%d]: expected content %q, got %q", i, content, backfillMsgs[i].Content)
		}
		if backfillMsgs[i].Type != message.TypeChat {
			t.Errorf("backfill[%d]: expected type 'chat', got %q", i, backfillMsgs[i].Type)
		}
	}
}

func TestHandlerNoBackfillForNewConnection(t *testing.T) {
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

	// New client joins — should NOT receive backfill (only new connections get session, not backfill).
	conn2 := dialAndJoin(t, ts.URL, "room1", "bob")
	defer conn2.Close(websocket.StatusNormalClosure, "")
	defer conn1.Close(websocket.StatusNormalClosure, "")

	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Bob should get "bob joined" system message, not a backfill.
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
	if env.Type == "backfill" {
		t.Error("new connections should not receive backfill")
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
