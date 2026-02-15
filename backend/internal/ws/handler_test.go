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

func newHandlerTestServer(t *testing.T, validateRoom RoomValidator) (*httptest.Server, *Hub) {
	t.Helper()
	hub := NewHub(nil)
	handler := NewHandler(hub, validateRoom)
	return httptest.NewServer(handler), hub
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
	return conn
}

func TestHandlerJoinAndChat(t *testing.T) {
	ts, hub := newHandlerTestServer(t, nil)
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
	ts, _ := newHandlerTestServer(t, func(roomID string) bool {
		return roomID == "valid-room"
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
	ts, _ := newHandlerTestServer(t, nil)
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
	ts, hub := newHandlerTestServer(t, nil)
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
