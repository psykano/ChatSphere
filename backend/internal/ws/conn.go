package ws

import (
	"context"
	"log"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

const (
	// sendBufferSize is the number of messages that can be queued per client.
	sendBufferSize = 16

	// writeTimeout is the max time to wait for a single write to complete.
	writeTimeout = 5 * time.Second
)

// ConnManager tracks all active WebSocket connections and provides
// lifecycle management including graceful shutdown and per-client
// buffered send channels.
type ConnManager struct {
	mu      sync.Mutex
	clients map[*Client]context.CancelFunc
	closed  bool
}

// NewConnManager creates a new connection manager.
func NewConnManager() *ConnManager {
	return &ConnManager{
		clients: make(map[*Client]context.CancelFunc),
	}
}

// Add registers a client and starts its write pump. The returned
// context is cancelled when the client is removed or the manager
// shuts down. Callers should select on ctx.Done() in their read loop.
func (cm *ConnManager) Add(c *Client) context.Context {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.closed {
		// Manager is shutting down; reject new connections.
		c.conn.Close(websocket.StatusGoingAway, "server shutting down")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	c.send = make(chan []byte, sendBufferSize)
	ctx, cancel := context.WithCancel(context.Background())
	cm.clients[c] = cancel

	go cm.writePump(ctx, c)

	return ctx
}

// Remove stops a client's write pump and cleans it up.
func (cm *ConnManager) Remove(c *Client) {
	cm.mu.Lock()
	cancel, ok := cm.clients[c]
	if ok {
		delete(cm.clients, c)
	}
	cm.mu.Unlock()

	if ok {
		cancel()
		close(c.send)
	}
}

// Send queues a message for delivery to the client. Returns false
// if the client's buffer is full (slow consumer) or the client has
// been removed.
func (cm *ConnManager) Send(c *Client, data []byte) bool {
	select {
	case c.send <- data:
		return true
	default:
		// Buffer full — drop the message to protect the server.
		log.Printf("ws: send buffer full for client %s, dropping message", c.userID)
		return false
	}
}

// Count returns the number of active connections.
func (cm *ConnManager) Count() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return len(cm.clients)
}

// Shutdown gracefully closes all connections. It cancels every write
// pump and closes each WebSocket with StatusGoingAway.
func (cm *ConnManager) Shutdown() {
	cm.mu.Lock()
	cm.closed = true
	clients := make(map[*Client]context.CancelFunc, len(cm.clients))
	for c, cancel := range cm.clients {
		clients[c] = cancel
	}
	cm.clients = make(map[*Client]context.CancelFunc)
	cm.mu.Unlock()

	for c, cancel := range clients {
		cancel()
		close(c.send)
		c.conn.Close(websocket.StatusGoingAway, "server shutting down")
	}
}

// writePump drains the client's send channel, writing each message
// to the WebSocket connection. It exits when ctx is cancelled or the
// send channel is closed.
func (cm *ConnManager) writePump(ctx context.Context, c *Client) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			if err := c.conn.Write(writeCtx, websocket.MessageText, msg); err != nil {
				cancel()
				log.Printf("ws: write to client %s failed: %v", c.userID, err)
				return
			}
			cancel()
		}
	}
}
