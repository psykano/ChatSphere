package message

import (
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestRedisStore(t *testing.T, maxSize int) (*RedisStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return NewRedisStore(client, maxSize), mr
}

func redisMsg(id, roomID, content string) *Message {
	return &Message{
		ID:        id,
		RoomID:    roomID,
		Content:   content,
		Type:      TypeChat,
		CreatedAt: time.Now(),
	}
}

func TestRedisStoreAppendAndCount(t *testing.T) {
	s, _ := newTestRedisStore(t, 100)

	s.Append(redisMsg("1", "room1", "hello"))
	s.Append(redisMsg("2", "room1", "world"))

	if s.Count("room1") != 2 {
		t.Fatalf("expected 2 messages, got %d", s.Count("room1"))
	}
	if s.Count("room2") != 0 {
		t.Fatalf("expected 0 messages for room2, got %d", s.Count("room2"))
	}
}

func TestRedisStoreMaxSize(t *testing.T) {
	s, _ := newTestRedisStore(t, 3)

	for i := 0; i < 5; i++ {
		s.Append(redisMsg(fmt.Sprintf("%d", i), "room1", fmt.Sprintf("msg-%d", i)))
	}

	if s.Count("room1") != 3 {
		t.Fatalf("expected 3 messages (max size), got %d", s.Count("room1"))
	}

	// Only IDs 2, 3, 4 remain. After "2" should give 3, 4.
	result := s.After("room1", "2")
	if len(result) != 2 {
		t.Fatalf("expected 2 messages after '2', got %d", len(result))
	}
	if result[0].ID != "3" || result[1].ID != "4" {
		t.Errorf("expected IDs [3, 4], got [%s, %s]", result[0].ID, result[1].ID)
	}
}

func TestRedisStoreAfterEmptyID(t *testing.T) {
	s, _ := newTestRedisStore(t, 100)
	s.Append(redisMsg("1", "room1", "hello"))

	result := s.After("room1", "")
	if len(result) != 0 {
		t.Fatalf("expected 0 messages for empty afterID, got %d", len(result))
	}
}

func TestRedisStoreAfterUnknownID(t *testing.T) {
	s, _ := newTestRedisStore(t, 100)
	s.Append(redisMsg("1", "room1", "hello"))
	s.Append(redisMsg("2", "room1", "world"))

	result := s.After("room1", "unknown")
	if result != nil {
		t.Fatalf("expected nil for unknown ID, got %d messages", len(result))
	}
}

func TestRedisStoreAfterLastMessage(t *testing.T) {
	s, _ := newTestRedisStore(t, 100)
	s.Append(redisMsg("1", "room1", "hello"))
	s.Append(redisMsg("2", "room1", "world"))

	result := s.After("room1", "2")
	if len(result) != 0 {
		t.Fatalf("expected 0 messages after last message, got %d", len(result))
	}
}

func TestRedisStoreAfterReturnsCorrectMessages(t *testing.T) {
	s, _ := newTestRedisStore(t, 100)
	s.Append(redisMsg("a", "room1", "first"))
	s.Append(redisMsg("b", "room1", "second"))
	s.Append(redisMsg("c", "room1", "third"))
	s.Append(redisMsg("d", "room1", "fourth"))

	result := s.After("room1", "b")
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].ID != "c" || result[1].ID != "d" {
		t.Errorf("expected IDs [c, d], got [%s, %s]", result[0].ID, result[1].ID)
	}
}

func TestRedisStoreRoomIsolation(t *testing.T) {
	s, _ := newTestRedisStore(t, 100)
	s.Append(redisMsg("1", "room1", "room1-msg"))
	s.Append(redisMsg("2", "room2", "room2-msg"))

	if s.Count("room1") != 1 {
		t.Errorf("expected 1 message in room1, got %d", s.Count("room1"))
	}
	if s.Count("room2") != 1 {
		t.Errorf("expected 1 message in room2, got %d", s.Count("room2"))
	}

	result := s.After("room1", "1")
	if len(result) != 0 {
		t.Errorf("expected 0 messages after '1' in room1, got %d", len(result))
	}
}

func TestRedisStoreDeleteRoom(t *testing.T) {
	s, _ := newTestRedisStore(t, 100)
	s.Append(redisMsg("1", "room1", "hello"))
	s.DeleteRoom("room1")

	if s.Count("room1") != 0 {
		t.Fatalf("expected 0 after delete, got %d", s.Count("room1"))
	}
}

func TestRedisStorePreservesMessageFields(t *testing.T) {
	s, _ := newTestRedisStore(t, 100)

	now := time.Now().Truncate(time.Second)
	s.Append(redisMsg("sentinel", "room1", "x"))
	s.Append(&Message{
		ID:        "target",
		RoomID:    "room1",
		UserID:    "user1",
		Username:  "alice",
		Content:   "hello world",
		Type:      TypeChat,
		CreatedAt: now,
	})

	result := s.After("room1", "sentinel")
	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}
	m := result[0]
	if m.ID != "target" {
		t.Errorf("expected ID 'target', got %q", m.ID)
	}
	if m.UserID != "user1" {
		t.Errorf("expected UserID 'user1', got %q", m.UserID)
	}
	if m.Username != "alice" {
		t.Errorf("expected Username 'alice', got %q", m.Username)
	}
	if m.Content != "hello world" {
		t.Errorf("expected Content 'hello world', got %q", m.Content)
	}
	if m.Type != TypeChat {
		t.Errorf("expected Type 'chat', got %q", m.Type)
	}
	if !m.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, m.CreatedAt)
	}
}

func TestRedisStoreDeleteRoomNonExistent(t *testing.T) {
	s, _ := newTestRedisStore(t, 100)
	// Should not panic or error.
	s.DeleteRoom("nonexistent")
}

func TestRedisStoreImplementsInterface(t *testing.T) {
	s, _ := newTestRedisStore(t, 100)
	var _ MessageStore = s
}
