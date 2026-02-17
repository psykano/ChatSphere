package user

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// AnonymousSession represents a persistent anonymous user identity.
// Unlike WebSocket sessions (which are per-room and per-connection),
// this provides a stable user ID across page reloads and room changes.
type AnonymousSession struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionStore manages anonymous user sessions keyed by token.
type SessionStore struct {
	mu       sync.Mutex
	sessions map[string]*AnonymousSession
}

// NewSessionStore creates a new anonymous session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*AnonymousSession),
	}
}

// Create generates a new anonymous session with a random token and user ID.
func (s *SessionStore) Create() *AnonymousSession {
	sess := &AnonymousSession{
		Token:     generateToken(),
		UserID:    generateToken(),
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	s.sessions[sess.Token] = sess
	s.mu.Unlock()
	return sess
}

// Get returns the session for the given token, or nil if not found.
func (s *SessionStore) Get(token string) *AnonymousSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[token]
}

// Count returns the number of sessions.
func (s *SessionStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
