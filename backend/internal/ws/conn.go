package ws

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"nhooyr.io/websocket"
)

const (
	// sendBufferSize is the number of messages that can be queued per client.
	sendBufferSize = 16

	// writeTimeout is the max time to wait for a single write to complete.
	writeTimeout = 5 * time.Second

	// defaultMaxConns is the default maximum concurrent connections (0 = unlimited).
	defaultMaxConns = 0

	// defaultIdleTimeout is the default time after which an idle connection is reaped.
	defaultIdleTimeout = 0

	// idleCheckInterval is how often the idle reaper runs.
	idleCheckInterval = 30 * time.Second
)

// connEntry holds per-connection metadata alongside the cancel function.
type connEntry struct {
	cancel      context.CancelFunc
	connectedAt time.Time
	lastActive  time.Time
}

// ConnStats holds point-in-time connection statistics.
type ConnStats struct {
	Active          int
	MaxConns        int
	Rejected        int64
	DroppedMessages int64
	IdleReaped      int64
}

// ConnManager tracks all active WebSocket connections and provides
// lifecycle management including graceful shutdown, per-client
// buffered send channels, connection limits, and idle detection.
type ConnManager struct {
	mu       sync.Mutex
	clients  map[*Client]*connEntry
	closed   bool
	maxConns int
	idleTTL  time.Duration
	stopIdle context.CancelFunc

	// Atomic counters for stats.
	rejected        atomic.Int64
	droppedMessages atomic.Int64
	idleReaped      atomic.Int64
}

// ConnManagerOption configures a ConnManager.
type ConnManagerOption func(*ConnManager)

// WithMaxConns sets the maximum number of concurrent connections.
// When the limit is reached, new connections are rejected.
// A value of 0 means unlimited (default).
func WithMaxConns(n int) ConnManagerOption {
	return func(cm *ConnManager) {
		cm.maxConns = n
	}
}

// WithIdleTimeout sets how long a connection can be idle before
// it is automatically closed. A value of 0 disables idle reaping (default).
func WithIdleTimeout(d time.Duration) ConnManagerOption {
	return func(cm *ConnManager) {
		cm.idleTTL = d
	}
}

// NewConnManager creates a new connection manager with optional configuration.
func NewConnManager(opts ...ConnManagerOption) *ConnManager {
	cm := &ConnManager{
		clients:  make(map[*Client]*connEntry),
		maxConns: defaultMaxConns,
		idleTTL:  defaultIdleTimeout,
	}
	for _, opt := range opts {
		opt(cm)
	}
	if cm.idleTTL > 0 {
		ctx, cancel := context.WithCancel(context.Background())
		cm.stopIdle = cancel
		go cm.idleReapLoop(ctx)
	}
	return cm
}

// Add registers a client and starts its write pump. The returned
// context is cancelled when the client is removed or the manager
// shuts down. Callers should select on ctx.Done() in their read loop.
// Returns a cancelled context if the manager is closed or at capacity.
func (cm *ConnManager) Add(c *Client) context.Context {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.closed {
		c.conn.Close(websocket.StatusGoingAway, "server shutting down")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	if cm.maxConns > 0 && len(cm.clients) >= cm.maxConns {
		cm.rejected.Add(1)
		c.conn.Close(websocket.StatusTryAgainLater, "server at capacity")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	now := time.Now()
	c.send = make(chan []byte, sendBufferSize)
	ctx, cancel := context.WithCancel(context.Background())
	cm.clients[c] = &connEntry{
		cancel:      cancel,
		connectedAt: now,
		lastActive:  now,
	}

	go cm.writePump(ctx, c)

	return ctx
}

// Remove stops a client's write pump and cleans it up.
func (cm *ConnManager) Remove(c *Client) {
	cm.mu.Lock()
	entry, ok := cm.clients[c]
	if ok {
		delete(cm.clients, c)
	}
	cm.mu.Unlock()

	if ok {
		entry.cancel()
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
		cm.droppedMessages.Add(1)
		log.Printf("ws: send buffer full for client %s, dropping message", c.userID)
		return false
	}
}

// TouchActivity updates the last-active timestamp for a client.
// Call this when a client sends a message to prevent idle reaping.
func (cm *ConnManager) TouchActivity(c *Client) {
	cm.mu.Lock()
	if entry, ok := cm.clients[c]; ok {
		entry.lastActive = time.Now()
	}
	cm.mu.Unlock()
}

// Count returns the number of active connections.
func (cm *ConnManager) Count() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return len(cm.clients)
}

// Stats returns point-in-time connection statistics.
func (cm *ConnManager) Stats() ConnStats {
	cm.mu.Lock()
	active := len(cm.clients)
	maxConns := cm.maxConns
	cm.mu.Unlock()
	return ConnStats{
		Active:          active,
		MaxConns:        maxConns,
		Rejected:        cm.rejected.Load(),
		DroppedMessages: cm.droppedMessages.Load(),
		IdleReaped:      cm.idleReaped.Load(),
	}
}

// ConnInfo holds metadata about a single connection.
type ConnInfo struct {
	UserID      string
	Username    string
	RoomID      string
	ConnectedAt time.Time
	LastActive  time.Time
	Idle        time.Duration
}

// Clients returns metadata for all active connections.
func (cm *ConnManager) Clients() []ConnInfo {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	now := time.Now()
	result := make([]ConnInfo, 0, len(cm.clients))
	for c, entry := range cm.clients {
		result = append(result, ConnInfo{
			UserID:      c.userID,
			Username:    c.username,
			RoomID:      c.roomID,
			ConnectedAt: entry.connectedAt,
			LastActive:  entry.lastActive,
			Idle:        now.Sub(entry.lastActive),
		})
	}
	return result
}

// Shutdown gracefully closes all connections. It cancels every write
// pump and closes each WebSocket with StatusGoingAway.
func (cm *ConnManager) Shutdown() {
	cm.mu.Lock()
	cm.closed = true
	clients := make(map[*Client]*connEntry, len(cm.clients))
	for c, entry := range cm.clients {
		clients[c] = entry
	}
	cm.clients = make(map[*Client]*connEntry)
	cm.mu.Unlock()

	if cm.stopIdle != nil {
		cm.stopIdle()
	}

	for c, entry := range clients {
		entry.cancel()
		close(c.send)
		c.conn.Close(websocket.StatusGoingAway, "server shutting down")
	}
}

// idleReapLoop periodically checks for and closes idle connections.
func (cm *ConnManager) idleReapLoop(ctx context.Context) {
	ticker := time.NewTicker(idleCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.reapIdle()
		}
	}
}

// reapIdle closes connections that have been idle longer than idleTTL.
func (cm *ConnManager) reapIdle() {
	cm.mu.Lock()
	now := time.Now()
	var stale []*Client
	for c, entry := range cm.clients {
		if now.Sub(entry.lastActive) > cm.idleTTL {
			stale = append(stale, c)
		}
	}
	// Remove stale entries while still holding the lock.
	entries := make(map[*Client]*connEntry, len(stale))
	for _, c := range stale {
		entries[c] = cm.clients[c]
		delete(cm.clients, c)
	}
	cm.mu.Unlock()

	for c, entry := range entries {
		entry.cancel()
		close(c.send)
		c.conn.Close(websocket.StatusPolicyViolation, "idle timeout")
		cm.idleReaped.Add(1)
		log.Printf("ws: reaped idle connection for client %s", c.userID)
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
