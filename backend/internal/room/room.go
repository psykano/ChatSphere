package room

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"
)

// Room represents a chat room.
type Room struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Capacity    int       `json:"capacity"`
	Public      bool      `json:"public"`
	Code        string    `json:"code,omitempty"`
	CreatorID   string    `json:"creator_id"`
	CreatedAt   time.Time `json:"created_at"`
	activeUsers atomic.Int32
	ActiveUsers int `json:"active_users"`

	mu             sync.Mutex
	lastMessageAt  time.Time
	lastUserLeftAt time.Time
	msgWarnSent    bool
	emptyWarnSent  bool
}

// AddActiveUsers atomically adjusts the active user count and syncs it
// to the exported field for JSON serialization.
func (r *Room) AddActiveUsers(delta int) {
	r.ActiveUsers = int(r.activeUsers.Add(int32(delta)))
}

// IsFull returns true if the room has reached its capacity.
func (r *Room) IsFull() bool {
	return int(r.activeUsers.Load()) >= r.Capacity
}

// TouchMessage records that a message was sent in this room.
func (r *Room) TouchMessage() {
	r.mu.Lock()
	r.lastMessageAt = time.Now()
	r.msgWarnSent = false
	r.mu.Unlock()
}

// TouchUserLeft records that a user left and the room became empty.
func (r *Room) TouchUserLeft() {
	r.mu.Lock()
	r.lastUserLeftAt = time.Now()
	r.mu.Unlock()
}

// ClearUserLeft clears the last-user-left timestamp (a user joined).
func (r *Room) ClearUserLeft() {
	r.mu.Lock()
	r.lastUserLeftAt = time.Time{}
	r.emptyWarnSent = false
	r.mu.Unlock()
}

// WarningReason describes why a room is about to expire.
type WarningReason int

const (
	WarnNone         WarningReason = iota
	WarnMsgInactive                // Room will expire due to message inactivity.
	WarnEmpty                      // Room will expire because it is empty.
)

// NeedsWarning reports whether the room is approaching expiration and a
// warning should be sent. It returns the reason and the time remaining
// until expiration. Each warning is sent at most once; the flag resets
// when activity resumes (new message or user join).
func (r *Room) NeedsWarning(msgTTL, msgWarn, emptyTTL, emptyWarn time.Duration, now time.Time) (WarningReason, time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check empty-room warning first (shorter TTL, more urgent).
	if !r.lastUserLeftAt.IsZero() && !r.emptyWarnSent {
		elapsed := now.Sub(r.lastUserLeftAt)
		remaining := emptyTTL - elapsed
		if remaining <= emptyWarn && remaining > 0 {
			r.emptyWarnSent = true
			return WarnEmpty, remaining
		}
	}

	// Check message inactivity warning.
	if !r.lastMessageAt.IsZero() && !r.msgWarnSent {
		elapsed := now.Sub(r.lastMessageAt)
		remaining := msgTTL - elapsed
		if remaining <= msgWarn && remaining > 0 {
			r.msgWarnSent = true
			return WarnMsgInactive, remaining
		}
	}

	return WarnNone, 0
}

// Expired reports whether the room should be reaped based on inactivity.
// A room expires if:
//   - No messages have been sent for msgTTL, OR
//   - No users have been present for emptyTTL.
func (r *Room) Expired(msgTTL, emptyTTL time.Duration, now time.Time) bool {
	r.mu.Lock()
	lastMsg := r.lastMessageAt
	lastLeft := r.lastUserLeftAt
	r.mu.Unlock()

	// Check empty-room expiration: room is empty and has been for emptyTTL.
	if !lastLeft.IsZero() && now.Sub(lastLeft) >= emptyTTL {
		return true
	}

	// Check message inactivity: no messages for msgTTL.
	if !lastMsg.IsZero() && now.Sub(lastMsg) >= msgTTL {
		return true
	}

	return false
}

// generateID returns a random hex ID.
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateCode returns a 6-character alphanumeric code for private rooms.
// Uses rejection sampling to avoid modulo bias.
func generateCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const maxUnbiased = 252 // largest multiple of 36 that fits in a byte (36*7=252)
	code := make([]byte, 6)
	buf := make([]byte, 12) // over-allocate to reduce Read calls
	for i := 0; i < 6; {
		rand.Read(buf)
		for _, b := range buf {
			if i >= 6 {
				break
			}
			if b < maxUnbiased {
				code[i] = charset[b%byte(len(charset))]
				i++
			}
		}
	}
	return string(code)
}

// uniqueCode generates a code that doesn't collide with existing rooms.
// Must be called while holding mu.
func (m *Manager) uniqueCode() string {
	for {
		code := generateCode()
		taken := false
		for _, r := range m.rooms {
			if r.Code == code {
				taken = true
				break
			}
		}
		if !taken {
			return code
		}
	}
}

// Manager manages chat rooms.
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room

	msgTTL   time.Duration
	emptyTTL time.Duration
	msgWarn  time.Duration
	emptyWarn time.Duration
	onExpire func(roomID string)
	onWarn   func(roomID string, reason WarningReason, remaining time.Duration)
}

// NewManager creates a new room Manager.
func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

// ExpirationConfig holds parameters for room expiration and warnings.
type ExpirationConfig struct {
	MsgTTL   time.Duration // How long without messages before expiring.
	EmptyTTL time.Duration // How long empty before expiring.
	MsgWarn  time.Duration // Warning window before message-inactivity expiration.
	EmptyWarn time.Duration // Warning window before empty-room expiration.
	OnExpire func(roomID string)
	OnWarn   func(roomID string, reason WarningReason, remaining time.Duration)
}

// StartExpiration begins a background goroutine that reaps expired rooms.
func (m *Manager) StartExpiration(cfg ExpirationConfig) {
	m.msgTTL = cfg.MsgTTL
	m.emptyTTL = cfg.EmptyTTL
	m.msgWarn = cfg.MsgWarn
	m.emptyWarn = cfg.EmptyWarn
	m.onExpire = cfg.OnExpire
	m.onWarn = cfg.OnWarn
	go m.reapLoop()
}

func (m *Manager) reapLoop() {
	interval := m.emptyTTL / 2
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		m.reap()
	}
}

type roomWarning struct {
	id        string
	reason    WarningReason
	remaining time.Duration
}

func (m *Manager) reap() {
	now := time.Now()
	m.mu.RLock()
	var expired []string
	var warnings []roomWarning
	for id, r := range m.rooms {
		if r.Expired(m.msgTTL, m.emptyTTL, now) {
			expired = append(expired, id)
			continue
		}
		if reason, remaining := r.NeedsWarning(m.msgTTL, m.msgWarn, m.emptyTTL, m.emptyWarn, now); reason != WarnNone {
			warnings = append(warnings, roomWarning{id, reason, remaining})
		}
	}
	m.mu.RUnlock()

	for _, w := range warnings {
		if m.onWarn != nil {
			m.onWarn(w.id, w.reason, w.remaining)
		}
	}

	for _, id := range expired {
		if m.onExpire != nil {
			m.onExpire(id)
		}
		m.Delete(id)
	}
}

// Create adds a new room and returns it.
func (m *Manager) Create(name, description, creatorID string, capacity int, public bool) *Room {
	now := time.Now()
	r := &Room{
		ID:            generateID(),
		Name:          name,
		Description:   description,
		Capacity:      capacity,
		Public:        public,
		CreatorID:     creatorID,
		CreatedAt:     now,
		lastMessageAt: now,
	}
	m.mu.Lock()
	if !public {
		r.Code = m.uniqueCode()
	}
	m.rooms[r.ID] = r
	m.mu.Unlock()

	return r
}

// Get returns a room by ID, or nil if not found.
func (m *Manager) Get(id string) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.rooms[id]
}

// GetByCode returns a private room matching the given code, or nil if not found.
func (m *Manager) GetByCode(code string) *Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.rooms {
		if !r.Public && r.Code == code {
			return r
		}
	}
	return nil
}

// List returns all public rooms sorted by active user count (descending).
func (m *Manager) List() []*Room {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Room
	for _, r := range m.rooms {
		if r.Public {
			result = append(result, r)
		}
	}

	// Sort by active users descending.
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].ActiveUsers > result[i].ActiveUsers {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// Delete removes a room by ID.
func (m *Manager) Delete(id string) {
	m.mu.Lock()
	delete(m.rooms, id)
	m.mu.Unlock()
}
