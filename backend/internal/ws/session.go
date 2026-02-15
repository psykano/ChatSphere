package ws

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Session holds the state needed to resume a WebSocket connection.
type Session struct {
	ID        string
	UserID    string
	Username  string
	RoomID    string
	CreatedAt time.Time

	// disconnectedAt is set when the client disconnects. A zero value
	// means the client is currently connected.
	disconnectedAt time.Time
}

// connected returns true if the session has an active connection.
func (s *Session) connected() bool {
	return s.disconnectedAt.IsZero()
}

// SessionStore manages sessions with expiration of disconnected sessions.
type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]*Session
	ttl      time.Duration
}

// NewSessionStore creates a store that expires disconnected sessions after ttl.
func NewSessionStore(ttl time.Duration) *SessionStore {
	ss := &SessionStore{
		sessions: make(map[string]*Session),
		ttl:      ttl,
	}
	go ss.reapLoop()
	return ss
}

// Create generates a new session for the given user and room.
func (ss *SessionStore) Create(userID, username, roomID string) *Session {
	id := generateSessionID()
	s := &Session{
		ID:        id,
		UserID:    userID,
		Username:  username,
		RoomID:    roomID,
		CreatedAt: time.Now(),
	}
	ss.mu.Lock()
	ss.sessions[id] = s
	ss.mu.Unlock()
	return s
}

// Get returns the session with the given ID, or nil if not found or expired.
func (ss *SessionStore) Get(id string) *Session {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.sessions[id]
}

// MarkDisconnected records the time the client disconnected. The session
// remains available for resumption until the TTL expires.
func (ss *SessionStore) MarkDisconnected(id string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if s, ok := ss.sessions[id]; ok {
		s.disconnectedAt = time.Now()
	}
}

// MarkConnected clears the disconnected timestamp, indicating the session
// has been resumed.
func (ss *SessionStore) MarkConnected(id string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if s, ok := ss.sessions[id]; ok {
		s.disconnectedAt = time.Time{}
	}
}

// Delete removes a session immediately.
func (ss *SessionStore) Delete(id string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.sessions, id)
}

// Count returns the number of sessions (both connected and disconnected).
func (ss *SessionStore) Count() int {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return len(ss.sessions)
}

// reapLoop periodically removes expired disconnected sessions.
func (ss *SessionStore) reapLoop() {
	ticker := time.NewTicker(ss.ttl / 2)
	defer ticker.Stop()
	for range ticker.C {
		ss.reap()
	}
}

// reap removes disconnected sessions older than the TTL.
func (ss *SessionStore) reap() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	now := time.Now()
	for id, s := range ss.sessions {
		if !s.connected() && now.Sub(s.disconnectedAt) > ss.ttl {
			delete(ss.sessions, id)
		}
	}
}

func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
