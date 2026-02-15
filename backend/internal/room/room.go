package room

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
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
	ActiveUsers int       `json:"active_users"`
}

// generateID returns a random hex ID.
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateCode returns a 6-character alphanumeric code for private rooms.
func generateCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	rand.Read(b)
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// Manager manages chat rooms.
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

// NewManager creates a new room Manager.
func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

// Create adds a new room and returns it.
func (m *Manager) Create(name, description, creatorID string, capacity int, public bool) *Room {
	r := &Room{
		ID:          generateID(),
		Name:        name,
		Description: description,
		Capacity:    capacity,
		Public:      public,
		CreatorID:   creatorID,
		CreatedAt:   time.Now(),
	}
	if !public {
		r.Code = generateCode()
	}

	m.mu.Lock()
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
