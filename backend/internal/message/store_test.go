package message

import (
	"fmt"
	"testing"
	"time"
)

func msg(id, roomID, content string) *Message {
	return &Message{
		ID:        id,
		RoomID:    roomID,
		Content:   content,
		Type:      TypeChat,
		CreatedAt: time.Now(),
	}
}

func TestStoreAppendAndCount(t *testing.T) {
	s := NewStore(100)

	s.Append(msg("1", "room1", "hello"))
	s.Append(msg("2", "room1", "world"))

	if s.Count("room1") != 2 {
		t.Fatalf("expected 2 messages, got %d", s.Count("room1"))
	}
	if s.Count("room2") != 0 {
		t.Fatalf("expected 0 messages for room2, got %d", s.Count("room2"))
	}
}

func TestStoreMaxSize(t *testing.T) {
	s := NewStore(3)

	for i := 0; i < 5; i++ {
		s.Append(msg(fmt.Sprintf("%d", i), "room1", fmt.Sprintf("msg-%d", i)))
	}

	if s.Count("room1") != 3 {
		t.Fatalf("expected 3 messages (max size), got %d", s.Count("room1"))
	}

	// After should return messages starting after ID "2" (which was evicted).
	// Only IDs 2, 3, 4 remain. After "2" should give 3, 4.
	result := s.After("room1", "2")
	if len(result) != 2 {
		t.Fatalf("expected 2 messages after '2', got %d", len(result))
	}
	if result[0].ID != "3" || result[1].ID != "4" {
		t.Errorf("expected IDs [3, 4], got [%s, %s]", result[0].ID, result[1].ID)
	}
}

func TestStoreAfterEmptyID(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("1", "room1", "hello"))

	result := s.After("room1", "")
	if len(result) != 0 {
		t.Fatalf("expected 0 messages for empty afterID, got %d", len(result))
	}
}

func TestStoreAfterUnknownID(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("1", "room1", "hello"))
	s.Append(msg("2", "room1", "world"))

	result := s.After("room1", "unknown")
	if result != nil {
		t.Fatalf("expected nil for unknown ID, got %d messages", len(result))
	}
}

func TestStoreAfterLastMessage(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("1", "room1", "hello"))
	s.Append(msg("2", "room1", "world"))

	result := s.After("room1", "2")
	if len(result) != 0 {
		t.Fatalf("expected 0 messages after last message, got %d", len(result))
	}
}

func TestStoreAfterReturnsCorrectMessages(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("a", "room1", "first"))
	s.Append(msg("b", "room1", "second"))
	s.Append(msg("c", "room1", "third"))
	s.Append(msg("d", "room1", "fourth"))

	result := s.After("room1", "b")
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].ID != "c" || result[1].ID != "d" {
		t.Errorf("expected IDs [c, d], got [%s, %s]", result[0].ID, result[1].ID)
	}
}

func TestStoreRoomIsolation(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("1", "room1", "room1-msg"))
	s.Append(msg("2", "room2", "room2-msg"))

	if s.Count("room1") != 1 {
		t.Errorf("expected 1 message in room1, got %d", s.Count("room1"))
	}
	if s.Count("room2") != 1 {
		t.Errorf("expected 1 message in room2, got %d", s.Count("room2"))
	}

	// After on room1 should not see room2 messages.
	result := s.After("room1", "1")
	if len(result) != 0 {
		t.Errorf("expected 0 messages after '1' in room1, got %d", len(result))
	}
}

func TestStoreDeleteRoom(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("1", "room1", "hello"))
	s.DeleteRoom("room1")

	if s.Count("room1") != 0 {
		t.Fatalf("expected 0 after delete, got %d", s.Count("room1"))
	}
}

func TestStoreRecentReturnsLastN(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("a", "room1", "first"))
	s.Append(msg("b", "room1", "second"))
	s.Append(msg("c", "room1", "third"))
	s.Append(msg("d", "room1", "fourth"))

	result := s.Recent("room1", 2)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].ID != "c" || result[1].ID != "d" {
		t.Errorf("expected IDs [c, d], got [%s, %s]", result[0].ID, result[1].ID)
	}
}

func TestStoreRecentFewerThanN(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("a", "room1", "first"))
	s.Append(msg("b", "room1", "second"))

	result := s.Recent("room1", 10)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].ID != "a" || result[1].ID != "b" {
		t.Errorf("expected IDs [a, b], got [%s, %s]", result[0].ID, result[1].ID)
	}
}

func TestStoreRecentEmptyRoom(t *testing.T) {
	s := NewStore(100)

	result := s.Recent("room1", 10)
	if result != nil {
		t.Fatalf("expected nil for empty room, got %d messages", len(result))
	}
}

func TestStoreRecentReturnsCopy(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("1", "room1", "first"))
	s.Append(msg("2", "room1", "second"))

	result := s.Recent("room1", 2)
	result[0] = msg("x", "room1", "modified")

	check := s.Recent("room1", 2)
	if check[0].ID != "1" {
		t.Errorf("store was mutated: expected ID '1', got %q", check[0].ID)
	}
}

func TestStoreBeforeReturnsOlderMessages(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("a", "room1", "first"))
	s.Append(msg("b", "room1", "second"))
	s.Append(msg("c", "room1", "third"))
	s.Append(msg("d", "room1", "fourth"))

	result := s.Before("room1", "d", 2)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].ID != "b" || result[1].ID != "c" {
		t.Errorf("expected IDs [b, c], got [%s, %s]", result[0].ID, result[1].ID)
	}
}

func TestStoreBeforeFirstMessage(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("a", "room1", "first"))
	s.Append(msg("b", "room1", "second"))

	result := s.Before("room1", "a", 5)
	if len(result) != 0 {
		t.Fatalf("expected 0 messages before first, got %d", len(result))
	}
}

func TestStoreBeforeEmptyID(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("a", "room1", "first"))

	result := s.Before("room1", "", 5)
	if result != nil {
		t.Fatalf("expected nil for empty beforeID, got %d messages", len(result))
	}
}

func TestStoreBeforeUnknownID(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("a", "room1", "first"))

	result := s.Before("room1", "unknown", 5)
	if result != nil {
		t.Fatalf("expected nil for unknown ID, got %d messages", len(result))
	}
}

func TestStoreBeforeFewerThanN(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("a", "room1", "first"))
	s.Append(msg("b", "room1", "second"))
	s.Append(msg("c", "room1", "third"))

	result := s.Before("room1", "c", 10)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}
	if result[0].ID != "a" || result[1].ID != "b" {
		t.Errorf("expected IDs [a, b], got [%s, %s]", result[0].ID, result[1].ID)
	}
}

func TestStoreBeforeReturnsCopy(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("a", "room1", "first"))
	s.Append(msg("b", "room1", "second"))
	s.Append(msg("c", "room1", "third"))

	result := s.Before("room1", "c", 2)
	result[0] = msg("x", "room1", "modified")

	check := s.Before("room1", "c", 2)
	if check[0].ID != "a" {
		t.Errorf("store was mutated: expected ID 'a', got %q", check[0].ID)
	}
}

func TestStoreAfterReturnsCopy(t *testing.T) {
	s := NewStore(100)
	s.Append(msg("1", "room1", "first"))
	s.Append(msg("2", "room1", "second"))
	s.Append(msg("3", "room1", "third"))

	result := s.After("room1", "1")
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	// Modifying the result should not affect the store.
	result[0] = msg("x", "room1", "modified")

	check := s.After("room1", "1")
	if check[0].ID != "2" {
		t.Errorf("store was mutated: expected ID '2', got %q", check[0].ID)
	}
}
