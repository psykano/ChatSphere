package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/christopherjohns/chatsphere/internal/message"
	"nhooyr.io/websocket"
)

// newConnTestServer creates a test server that registers each connection
// with the given hub and room, then reads until the connection closes.
func newConnTestServer(t *testing.T, hub *Hub, roomID string) *httptest.Server {
	t.Helper()
	var counter atomic.Int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept error: %v", err)
			return
		}

		n := counter.Add(1)
		client := &Client{
			conn:     conn,
			userID:   "user-" + string(rune('a'-1+n)),
			username: "tester",
			roomID:   roomID,
			hub:      hub,
		}
		connCtx := hub.addClient(client)
		defer hub.removeClient(client)

		// Read until closed or context cancelled.
		for {
			select {
			case <-connCtx.Done():
				return
			default:
			}
			_, _, err := conn.Read(r.Context())
			if err != nil {
				return
			}
		}
	}))
}

func TestConnManagerAddRemove(t *testing.T) {
	cm := NewConnManager()

	client := &Client{userID: "test-1"}
	// Simulate a minimal conn by using a real WebSocket pair.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		client.conn = conn
		// Block until test closes.
		for {
			_, _, err := conn.Read(r.Context())
			if err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	wsConn := dialWS(t, ts.URL)
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	// Wait for server handler to set client.conn.
	deadline := time.Now().Add(2 * time.Second)
	for client.conn == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if client.conn == nil {
		t.Fatal("client.conn was not set")
	}

	ctx := cm.Add(client)
	if cm.Count() != 1 {
		t.Fatalf("expected 1 connection, got %d", cm.Count())
	}
	if client.send == nil {
		t.Fatal("expected send channel to be initialized")
	}

	// Context should not be cancelled.
	select {
	case <-ctx.Done():
		t.Fatal("context should not be cancelled yet")
	default:
	}

	cm.Remove(client)
	if cm.Count() != 0 {
		t.Fatalf("expected 0 connections after remove, got %d", cm.Count())
	}

	// Context should be cancelled after remove.
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("context should be cancelled after remove")
	}
}

func TestConnManagerSend(t *testing.T) {
	hub := NewHub(nil)

	ts := newConnTestServer(t, hub, "room1")
	defer ts.Close()

	conn := dialWS(t, ts.URL)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Wait for registration.
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.ClientCount("room1") != 1 {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount("room1"))
	}

	// Broadcast a message and verify it arrives via the send channel / write pump.
	hub.Broadcast("room1", &message.Message{
		ID:      "msg1",
		RoomID:  "room1",
		Content: "hello via conn manager",
		Type:    message.TypeChat,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if env.Type != "chat" {
		t.Errorf("expected type 'chat', got %q", env.Type)
	}

	var msg message.Message
	if err := json.Unmarshal(env.Payload, &msg); err != nil {
		t.Fatalf("unmarshal payload error: %v", err)
	}
	if msg.Content != "hello via conn manager" {
		t.Errorf("expected 'hello via conn manager', got %q", msg.Content)
	}
}

func TestConnManagerSendBufferFull(t *testing.T) {
	cm := NewConnManager()

	client := &Client{userID: "slow-consumer"}
	// We don't need a real connection for this test — just the send channel.
	client.send = make(chan []byte, sendBufferSize)
	// Manually add to track in manager.
	now := time.Now()
	cm.mu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	cm.clients[client] = &connEntry{cancel: cancel, connectedAt: now, lastActive: now}
	cm.mu.Unlock()
	defer func() {
		cancel()
		ctx.Done() // use ctx to avoid unused warning
	}()

	// Fill the buffer.
	for i := 0; i < sendBufferSize; i++ {
		if !cm.Send(client, []byte("msg")) {
			t.Fatalf("send %d should have succeeded", i)
		}
	}

	// Next send should fail (buffer full).
	if cm.Send(client, []byte("overflow")) {
		t.Fatal("expected send to fail when buffer is full")
	}
}

func TestConnManagerConcurrentSend(t *testing.T) {
	hub := NewHub(nil)

	ts := newConnTestServer(t, hub, "room1")
	defer ts.Close()

	// Connect multiple clients.
	const numClients = 5
	conns := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conns[i] = dialWS(t, ts.URL)
		defer conns[i].Close(websocket.StatusNormalClosure, "")
	}

	// Wait for all to register.
	deadline := time.Now().Add(3 * time.Second)
	for hub.ClientCount("room1") < numClients && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.ClientCount("room1") != numClients {
		t.Fatalf("expected %d clients, got %d", numClients, hub.ClientCount("room1"))
	}

	// Send messages concurrently.
	const numMessages = 10
	var wg sync.WaitGroup
	for i := 0; i < numMessages; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.Broadcast("room1", &message.Message{
				ID:      generateClientID(),
				RoomID:  "room1",
				Content: "concurrent",
				Type:    message.TypeChat,
			})
		}()
	}
	wg.Wait()

	// Each client should receive all messages.
	for ci, conn := range conns {
		for mi := 0; mi < numMessages; mi++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, _, err := conn.Read(ctx)
			cancel()
			if err != nil {
				t.Fatalf("client %d: read message %d error: %v", ci, mi, err)
			}
		}
	}
}

func TestConnManagerShutdown(t *testing.T) {
	hub := NewHub(nil)

	ts := newConnTestServer(t, hub, "room1")
	defer ts.Close()

	conn := dialWS(t, ts.URL)
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Wait for registration.
	deadline := time.Now().Add(2 * time.Second)
	for hub.ClientCount("room1") == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if hub.ConnMgr().Count() != 1 {
		t.Fatalf("expected 1 managed connection, got %d", hub.ConnMgr().Count())
	}

	// Shutdown the connection manager.
	hub.ConnMgr().Shutdown()

	if hub.ConnMgr().Count() != 0 {
		t.Fatalf("expected 0 connections after shutdown, got %d", hub.ConnMgr().Count())
	}

	// The WebSocket should be closed — reads should fail.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _, err := conn.Read(ctx)
	if err == nil {
		t.Fatal("expected read to fail after shutdown")
	}
}

func TestConnManagerShutdownRejectsNew(t *testing.T) {
	cm := NewConnManager()
	cm.Shutdown()

	// After shutdown, Add should reject new connections.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		client := &Client{
			conn:   conn,
			userID: "late",
		}
		ctx := cm.Add(client)
		// Context should be immediately cancelled.
		select {
		case <-ctx.Done():
		default:
			t.Error("expected context to be cancelled for rejected client")
		}
	}))
	defer ts.Close()

	wsConn := dialWS(t, ts.URL)
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	// Give the server handler time to execute.
	time.Sleep(100 * time.Millisecond)

	if cm.Count() != 0 {
		t.Fatalf("expected 0 connections after shutdown, got %d", cm.Count())
	}
}

func TestConnManagerDoubleRemove(t *testing.T) {
	cm := NewConnManager()

	client := &Client{userID: "test-double"}
	client.send = make(chan []byte, sendBufferSize)

	now := time.Now()
	cm.mu.Lock()
	_, cancel := context.WithCancel(context.Background())
	cm.clients[client] = &connEntry{cancel: cancel, connectedAt: now, lastActive: now}
	cm.mu.Unlock()

	// First remove should work.
	cm.Remove(client)
	if cm.Count() != 0 {
		t.Fatalf("expected 0, got %d", cm.Count())
	}

	// Second remove should be a no-op (no panic).
	cm.Remove(client)
}

func TestConnManagerMaxConns(t *testing.T) {
	cm := NewConnManager(WithMaxConns(2))

	var clients [3]*Client
	var servers [3]*httptest.Server

	for i := range 3 {
		clients[i] = &Client{userID: "user-" + string(rune('a'+i))}
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, nil)
			if err != nil {
				return
			}
			clients[i].conn = conn
			for {
				_, _, err := conn.Read(r.Context())
				if err != nil {
					return
				}
			}
		}))
		defer servers[i].Close()
	}

	// Connect all three — set up their conn fields.
	wsConns := make([]*websocket.Conn, 3)
	for i := range 3 {
		wsConns[i] = dialWS(t, servers[i].URL)
		defer wsConns[i].Close(websocket.StatusNormalClosure, "")
		deadline := time.Now().Add(2 * time.Second)
		for clients[i].conn == nil && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// First two should succeed.
	ctx0 := cm.Add(clients[0])
	ctx1 := cm.Add(clients[1])
	if cm.Count() != 2 {
		t.Fatalf("expected 2, got %d", cm.Count())
	}

	// Third should be rejected.
	ctx2 := cm.Add(clients[2])
	select {
	case <-ctx2.Done():
		// Expected — context cancelled immediately.
	default:
		t.Fatal("expected third connection to be rejected")
	}
	if cm.Count() != 2 {
		t.Fatalf("expected 2 after rejection, got %d", cm.Count())
	}
	if cm.Stats().Rejected != 1 {
		t.Fatalf("expected 1 rejected, got %d", cm.Stats().Rejected)
	}

	// Remove one, then third should succeed.
	cm.Remove(clients[0])
	_ = ctx0

	clients[2].conn = nil // Reset to re-dial.
	servers[2].Close()
	servers[2] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		clients[2].conn = conn
		for {
			_, _, err := conn.Read(r.Context())
			if err != nil {
				return
			}
		}
	}))
	defer servers[2].Close()
	wsConns[2].Close(websocket.StatusNormalClosure, "")
	wsConns[2] = dialWS(t, servers[2].URL)
	defer wsConns[2].Close(websocket.StatusNormalClosure, "")
	deadline := time.Now().Add(2 * time.Second)
	for clients[2].conn == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	ctx2 = cm.Add(clients[2])
	select {
	case <-ctx2.Done():
		t.Fatal("expected third connection to succeed after slot freed")
	default:
	}
	if cm.Count() != 2 {
		t.Fatalf("expected 2, got %d", cm.Count())
	}

	_ = ctx1
	cm.Shutdown()
}

func TestConnManagerStats(t *testing.T) {
	cm := NewConnManager(WithMaxConns(1))

	stats := cm.Stats()
	if stats.Active != 0 {
		t.Fatalf("expected 0 active, got %d", stats.Active)
	}
	if stats.MaxConns != 1 {
		t.Fatalf("expected maxConns=1, got %d", stats.MaxConns)
	}

	client := &Client{userID: "stat-test"}
	client.send = make(chan []byte, sendBufferSize)

	now := time.Now()
	cm.mu.Lock()
	_, cancel := context.WithCancel(context.Background())
	cm.clients[client] = &connEntry{cancel: cancel, connectedAt: now, lastActive: now}
	cm.mu.Unlock()

	stats = cm.Stats()
	if stats.Active != 1 {
		t.Fatalf("expected 1 active, got %d", stats.Active)
	}

	// Fill the buffer and trigger dropped messages.
	for i := 0; i < sendBufferSize; i++ {
		cm.Send(client, []byte("msg"))
	}
	cm.Send(client, []byte("overflow"))

	stats = cm.Stats()
	if stats.DroppedMessages != 1 {
		t.Fatalf("expected 1 dropped, got %d", stats.DroppedMessages)
	}

	cancel()
	cm.Remove(client)
}

func TestConnManagerTouchActivity(t *testing.T) {
	cm := NewConnManager()

	client := &Client{userID: "active-test"}
	client.send = make(chan []byte, sendBufferSize)

	past := time.Now().Add(-10 * time.Minute)
	cm.mu.Lock()
	_, cancel := context.WithCancel(context.Background())
	cm.clients[client] = &connEntry{cancel: cancel, connectedAt: past, lastActive: past}
	cm.mu.Unlock()
	defer func() {
		cancel()
	}()

	// Verify initial last-active is in the past.
	cm.mu.Lock()
	entry := cm.clients[client]
	initialActive := entry.lastActive
	cm.mu.Unlock()
	if !initialActive.Equal(past) {
		t.Fatalf("expected lastActive=%v, got %v", past, initialActive)
	}

	// Touch activity.
	cm.TouchActivity(client)

	cm.mu.Lock()
	entry = cm.clients[client]
	updatedActive := entry.lastActive
	cm.mu.Unlock()
	if !updatedActive.After(past) {
		t.Fatal("expected lastActive to be updated after TouchActivity")
	}
}

func TestConnManagerClients(t *testing.T) {
	cm := NewConnManager()

	c1 := &Client{userID: "u1", username: "alice", roomID: "room-a"}
	c1.send = make(chan []byte, sendBufferSize)
	c2 := &Client{userID: "u2", username: "bob", roomID: "room-b"}
	c2.send = make(chan []byte, sendBufferSize)

	now := time.Now()
	cm.mu.Lock()
	_, cancel1 := context.WithCancel(context.Background())
	cm.clients[c1] = &connEntry{cancel: cancel1, connectedAt: now, lastActive: now}
	_, cancel2 := context.WithCancel(context.Background())
	cm.clients[c2] = &connEntry{cancel: cancel2, connectedAt: now, lastActive: now}
	cm.mu.Unlock()
	defer func() {
		cancel1()
		cancel2()
	}()

	infos := cm.Clients()
	if len(infos) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(infos))
	}

	found := map[string]bool{}
	for _, info := range infos {
		found[info.UserID] = true
		if info.ConnectedAt.IsZero() {
			t.Fatalf("expected non-zero ConnectedAt for %s", info.UserID)
		}
	}
	if !found["u1"] || !found["u2"] {
		t.Fatalf("expected both u1 and u2 in client list, got %v", found)
	}
}

func TestConnManagerIdleReap(t *testing.T) {
	cm := NewConnManager(WithIdleTimeout(100 * time.Millisecond))
	defer cm.Shutdown()

	// Create a client with a real WebSocket for the reaper to close.
	client := &Client{userID: "idle-test"}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		client.conn = conn
		for {
			_, _, err := conn.Read(r.Context())
			if err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	wsConn := dialWS(t, ts.URL)
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	deadline := time.Now().Add(2 * time.Second)
	for client.conn == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if client.conn == nil {
		t.Fatal("client.conn was not set")
	}

	// Set lastActive far in the past so it's immediately idle.
	past := time.Now().Add(-1 * time.Hour)
	client.send = make(chan []byte, sendBufferSize)
	cm.mu.Lock()
	_, cancel := context.WithCancel(context.Background())
	cm.clients[client] = &connEntry{cancel: cancel, connectedAt: past, lastActive: past}
	cm.mu.Unlock()

	if cm.Count() != 1 {
		t.Fatalf("expected 1 before reap, got %d", cm.Count())
	}

	// Trigger reap directly (don't wait for the background loop).
	cm.reapIdle()

	if cm.Count() != 0 {
		t.Fatalf("expected 0 after reap, got %d", cm.Count())
	}
	if cm.Stats().IdleReaped != 1 {
		t.Fatalf("expected 1 idle reaped, got %d", cm.Stats().IdleReaped)
	}
}

func TestConnManagerIdleReapSkipsActive(t *testing.T) {
	cm := NewConnManager(WithIdleTimeout(1 * time.Hour))
	defer cm.Shutdown()

	client := &Client{userID: "active-client"}
	client.send = make(chan []byte, sendBufferSize)

	now := time.Now()
	cm.mu.Lock()
	_, cancel := context.WithCancel(context.Background())
	cm.clients[client] = &connEntry{cancel: cancel, connectedAt: now, lastActive: now}
	cm.mu.Unlock()

	cm.reapIdle()

	if cm.Count() != 1 {
		t.Fatalf("expected 1 (active client should not be reaped), got %d", cm.Count())
	}
	if cm.Stats().IdleReaped != 0 {
		t.Fatalf("expected 0 idle reaped, got %d", cm.Stats().IdleReaped)
	}

	cancel()
	cm.Remove(client)
}

func TestConnManagerDroppedMessageStats(t *testing.T) {
	cm := NewConnManager()

	client := &Client{userID: "drop-test"}
	client.send = make(chan []byte, sendBufferSize)

	now := time.Now()
	cm.mu.Lock()
	_, cancel := context.WithCancel(context.Background())
	cm.clients[client] = &connEntry{cancel: cancel, connectedAt: now, lastActive: now}
	cm.mu.Unlock()
	defer func() {
		cancel()
	}()

	// Fill buffer.
	for i := 0; i < sendBufferSize; i++ {
		cm.Send(client, []byte("msg"))
	}

	// Drop 3 messages.
	for i := 0; i < 3; i++ {
		cm.Send(client, []byte("overflow"))
	}

	stats := cm.Stats()
	if stats.DroppedMessages != 3 {
		t.Fatalf("expected 3 dropped, got %d", stats.DroppedMessages)
	}
}
