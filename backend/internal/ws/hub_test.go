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
