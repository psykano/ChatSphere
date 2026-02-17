package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/christopherjohns/chatsphere/internal/message"
	"nhooyr.io/websocket"
)

// newTestServer starts an httptest.Server that upgrades to WebSocket and
// registers the connection in the hub under the given roomID.
func newTestServer(t *testing.T, hub *Hub, roomID string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept error: %v", err)
			return
		}

		client := &Client{
			conn:     conn,
			userID:   "test-user",
			username: "tester",
			roomID:   roomID,
			hub:      hub,
		}
		hub.addClient(client)
		defer hub.removeClient(client)

		// Keep reading to hold the connection open.
		for {
			_, _, err := conn.Read(r.Context())
			if err != nil {
				return
			}
		}
	}))
}

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(url, "http")
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	return conn
}

func TestHubAddRemoveClient(t *testing.T) {
	var joins atomic.Int32
	hub := NewHub(func(roomID string, delta int) {
		joins.Add(int32(delta))
	})

	ts := newTestServer(t, hub, "room1")
	defer ts.Close()

	conn := dialWS(t, ts.URL)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Wait for the server to register the client.
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if hub.ClientCount("room1") != 1 {
		t.Fatalf("expected 1 client in room1, got %d", hub.ClientCount("room1"))
	}
	if joins.Load() != 1 {
		t.Fatalf("expected onJoin called with +1, got %d", joins.Load())
	}

	conn.Close(websocket.StatusNormalClosure, "")
	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if hub.ClientCount("room1") != 0 {
		t.Fatalf("expected 0 clients after disconnect, got %d", hub.ClientCount("room1"))
	}
	if joins.Load() != 0 {
		t.Fatalf("expected net joins to be 0, got %d", joins.Load())
	}
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub(nil)

	ts := newTestServer(t, hub, "room1")
	defer ts.Close()

	conn := dialWS(t, ts.URL)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Wait for registration.
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Broadcast to room1.
	hub.Broadcast("room1", &message.Message{
		ID:      "msg1",
		RoomID:  "room1",
		Content: "hello",
		Type:    message.TypeChat,
	})

	// Read the broadcast message.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal envelope error: %v", err)
	}
	if env.Type != "chat" {
		t.Errorf("expected type 'chat', got %q", env.Type)
	}
}

func TestHubBroadcastIsolation(t *testing.T) {
	hub := NewHub(nil)

	ts1 := newTestServer(t, hub, "room1")
	defer ts1.Close()
	ts2 := newTestServer(t, hub, "room2")
	defer ts2.Close()

	conn1 := dialWS(t, ts1.URL)
	defer conn1.Close(websocket.StatusNormalClosure, "")
	conn2 := dialWS(t, ts2.URL)
	defer conn2.Close(websocket.StatusNormalClosure, "")

	// Wait for both connections.
	deadline := time.Now().Add(2 * time.Second)
	for (hub.ClientCount("room1") == 0 || hub.ClientCount("room2") == 0) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Broadcast only to room1.
	hub.Broadcast("room1", &message.Message{
		ID:      "msg1",
		RoomID:  "room1",
		Content: "for room1 only",
		Type:    message.TypeChat,
	})

	// conn1 should receive the message.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := conn1.Read(ctx)
	if err != nil {
		t.Fatalf("conn1 read error: %v", err)
	}

	// conn2 should NOT receive anything (expect timeout).
	ctx2, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	_, _, err = conn2.Read(ctx2)
	if err == nil {
		t.Fatal("conn2 should not have received a message for room1")
	}
}

func TestHubClientCountEmpty(t *testing.T) {
	hub := NewHub(nil)
	if hub.ClientCount("nonexistent") != 0 {
		t.Error("expected 0 for nonexistent room")
	}
}

func TestHubOnBroadcastCallback(t *testing.T) {
	var called atomic.Int32
	hub := NewHub(nil)
	hub.SetOnBroadcast(func(roomID string) {
		if roomID == "room1" {
			called.Add(1)
		}
	})

	ts := newTestServer(t, hub, "room1")
	defer ts.Close()

	conn := dialWS(t, ts.URL)
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	hub.Broadcast("room1", &message.Message{
		ID:      "msg1",
		RoomID:  "room1",
		Content: "hello",
		Type:    message.TypeChat,
	})

	time.Sleep(50 * time.Millisecond)

	if called.Load() != 1 {
		t.Errorf("expected onBroadcast called once, got %d", called.Load())
	}
}

func TestHubDisconnectRoom(t *testing.T) {
	hub := NewHub(nil)

	ts := newTestServer(t, hub, "room1")
	defer ts.Close()

	conn := dialWS(t, ts.URL)
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if hub.ClientCount("room1") != 1 {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount("room1"))
	}

	hub.DisconnectRoom("room1")

	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if hub.ClientCount("room1") != 0 {
		t.Errorf("expected 0 clients after DisconnectRoom, got %d", hub.ClientCount("room1"))
	}
}

func TestHubDisconnectRoomNoClients(t *testing.T) {
	hub := NewHub(nil)
	// Should not panic on empty room.
	hub.DisconnectRoom("nonexistent")
}

func TestHubRoomUsersEmpty(t *testing.T) {
	hub := NewHub(nil)
	users := hub.RoomUsers("nonexistent")
	if len(users) != 0 {
		t.Errorf("expected 0 users for nonexistent room, got %d", len(users))
	}
}

func TestHubRoomUsers(t *testing.T) {
	hub := NewHub(nil)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept error: %v", err)
			return
		}

		userID := r.URL.Query().Get("user_id")
		username := r.URL.Query().Get("username")
		client := &Client{
			conn:     conn,
			userID:   userID,
			username: username,
			roomID:   "room1",
			hub:      hub,
		}
		hub.addClient(client)
		defer hub.removeClient(client)

		for {
			_, _, err := conn.Read(r.Context())
			if err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	// Connect two clients with different user IDs.
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel1()
	conn1, _, err := websocket.Dial(ctx1, wsURL+"?user_id=u1&username=alice", nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn1.Close(websocket.StatusNormalClosure, "")

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	conn2, _, err := websocket.Dial(ctx2, wsURL+"?user_id=u2&username=bob", nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn2.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	users := hub.RoomUsers("room1")
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	found := map[string]string{}
	for _, u := range users {
		found[u.UserID] = u.Username
	}
	if found["u1"] != "alice" {
		t.Errorf("expected alice for u1, got %q", found["u1"])
	}
	if found["u2"] != "bob" {
		t.Errorf("expected bob for u2, got %q", found["u2"])
	}

	// Disconnect one client and verify the list updates.
	conn1.Close(websocket.StatusNormalClosure, "")
	deadline = time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") != 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	users = hub.RoomUsers("room1")
	if len(users) != 1 {
		t.Fatalf("expected 1 user after disconnect, got %d", len(users))
	}
	if users[0].UserID != "u2" || users[0].Username != "bob" {
		t.Errorf("expected bob to remain, got %+v", users[0])
	}
}

func TestHubBroadcastPresence(t *testing.T) {
	hub := NewHub(nil)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept error: %v", err)
			return
		}

		client := &Client{
			conn:     conn,
			userID:   r.URL.Query().Get("user_id"),
			username: r.URL.Query().Get("username"),
			roomID:   "room1",
			hub:      hub,
		}
		hub.addClient(client)
		defer hub.removeClient(client)

		for {
			_, _, err := conn.Read(r.Context())
			if err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, wsURL+"?user_id=u1&username=alice", nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	hub.BroadcastPresence("room1")

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
	if env.Type != "presence" {
		t.Fatalf("expected type 'presence', got %q", env.Type)
	}

	var payload PresencePayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		t.Fatalf("unmarshal presence error: %v", err)
	}
	if len(payload.Users) != 1 {
		t.Fatalf("expected 1 user in presence, got %d", len(payload.Users))
	}
	if payload.Users[0].UserID != "u1" || payload.Users[0].Username != "alice" {
		t.Errorf("unexpected user: %+v", payload.Users[0])
	}
}
