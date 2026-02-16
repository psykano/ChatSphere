package message

import "sync"

// MessageStore is the interface for message persistence backends.
type MessageStore interface {
	Append(msg *Message)
	After(roomID, afterID string) []*Message
	DeleteRoom(roomID string)
	Count(roomID string) int
}

// Store keeps recent messages per room in memory for backfill on reconnect.
type Store struct {
	mu      sync.RWMutex
	rooms   map[string][]*Message
	maxSize int
}

// NewStore creates a message store that retains up to maxSize messages per room.
func NewStore(maxSize int) *Store {
	return &Store{
		rooms:   make(map[string][]*Message),
		maxSize: maxSize,
	}
}

// Append adds a message to the room's history.
func (s *Store) Append(msg *Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msgs := s.rooms[msg.RoomID]
	msgs = append(msgs, msg)
	if len(msgs) > s.maxSize {
		msgs = msgs[len(msgs)-s.maxSize:]
	}
	s.rooms[msg.RoomID] = msgs
}

// After returns all messages in a room that were stored after the message
// with the given ID. If afterID is empty, no messages are returned.
func (s *Store) After(roomID, afterID string) []*Message {
	if afterID == "" {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := s.rooms[roomID]
	for i, m := range msgs {
		if m.ID == afterID {
			// Return everything after this index.
			result := make([]*Message, len(msgs)-i-1)
			copy(result, msgs[i+1:])
			return result
		}
	}
	return nil
}

// DeleteRoom removes all stored messages for a room.
func (s *Store) DeleteRoom(roomID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rooms, roomID)
}

// Count returns the number of stored messages for a room.
func (s *Store) Count(roomID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rooms[roomID])
}
