package room

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestManagerCreateAndGet(t *testing.T) {
	m := NewManager()
	r := m.Create("test-room", "A test room", "user1", 50, true)

	if r.Name != "test-room" {
		t.Errorf("expected name 'test-room', got %q", r.Name)
	}
	if r.Description != "A test room" {
		t.Errorf("expected description 'A test room', got %q", r.Description)
	}
	if r.Capacity != 50 {
		t.Errorf("expected capacity 50, got %d", r.Capacity)
	}
	if !r.Public {
		t.Error("expected room to be public")
	}
	if r.Code != "" {
		t.Errorf("expected no code for public room, got %q", r.Code)
	}

	got := m.Get(r.ID)
	if got == nil {
		t.Fatal("expected to find room by ID")
	}
	if got.ID != r.ID {
		t.Errorf("expected ID %q, got %q", r.ID, got.ID)
	}
}

func TestManagerCreatePrivateRoom(t *testing.T) {
	m := NewManager()
	r := m.Create("secret", "", "user1", 10, false)

	if r.Public {
		t.Error("expected room to be private")
	}
	if len(r.Code) != 6 {
		t.Errorf("expected 6-char code, got %q (len %d)", r.Code, len(r.Code))
	}
}

func TestManagerGetNotFound(t *testing.T) {
	m := NewManager()
	if m.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent room")
	}
}

func TestManagerListReturnsOnlyPublicRooms(t *testing.T) {
	m := NewManager()
	m.Create("public1", "", "user1", 50, true)
	m.Create("private1", "", "user1", 10, false)
	m.Create("public2", "", "user1", 50, true)

	rooms := m.List()
	if len(rooms) != 2 {
		t.Fatalf("expected 2 public rooms, got %d", len(rooms))
	}
	for _, r := range rooms {
		if !r.Public {
			t.Errorf("List() returned private room %q", r.Name)
		}
	}
}

func TestManagerListSortedByActiveUsers(t *testing.T) {
	m := NewManager()
	r1 := m.Create("low", "", "user1", 50, true)
	r2 := m.Create("high", "", "user1", 50, true)
	r3 := m.Create("mid", "", "user1", 50, true)

	r1.ActiveUsers = 1
	r2.ActiveUsers = 10
	r3.ActiveUsers = 5

	rooms := m.List()
	if len(rooms) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(rooms))
	}
	if rooms[0].ActiveUsers != 10 {
		t.Errorf("expected first room to have 10 active users, got %d", rooms[0].ActiveUsers)
	}
	if rooms[1].ActiveUsers != 5 {
		t.Errorf("expected second room to have 5 active users, got %d", rooms[1].ActiveUsers)
	}
	if rooms[2].ActiveUsers != 1 {
		t.Errorf("expected third room to have 1 active user, got %d", rooms[2].ActiveUsers)
	}
}

func TestManagerDelete(t *testing.T) {
	m := NewManager()
	r := m.Create("to-delete", "", "user1", 50, true)

	m.Delete(r.ID)
	if m.Get(r.ID) != nil {
		t.Error("expected room to be deleted")
	}
}

func TestManagerGetByCode(t *testing.T) {
	m := NewManager()
	priv := m.Create("secret", "", "user1", 10, false)
	m.Create("public", "", "user1", 50, true)

	got := m.GetByCode(priv.Code)
	if got == nil {
		t.Fatal("expected to find private room by code")
	}
	if got.ID != priv.ID {
		t.Errorf("expected room ID %q, got %q", priv.ID, got.ID)
	}
}

func TestManagerGetByCodeNotFound(t *testing.T) {
	m := NewManager()
	m.Create("secret", "", "user1", 10, false)

	if m.GetByCode("ZZZZZZ") != nil {
		t.Error("expected nil for non-matching code")
	}
}

func TestManagerUniqueCodeNoDuplicates(t *testing.T) {
	m := NewManager()
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		r := m.Create("room", "", "user1", 10, false)
		if seen[r.Code] {
			t.Fatalf("duplicate code %q generated", r.Code)
		}
		seen[r.Code] = true
	}
}

func TestGenerateCode(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := generateCode()
		if len(code) != 6 {
			t.Fatalf("expected 6-char code, got %q", code)
		}
		for _, c := range code {
			if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				t.Fatalf("invalid character %c in code %q", c, code)
			}
		}
		seen[code] = true
	}
	// With 36^6 possibilities, 100 codes should all be unique.
	if len(seen) != 100 {
		t.Errorf("expected 100 unique codes, got %d", len(seen))
	}
}

func TestRoomExpiredByMessageInactivity(t *testing.T) {
	m := NewManager()
	r := m.Create("test", "", "user1", 50, true)

	msgTTL := 2 * time.Hour
	emptyTTL := 15 * time.Minute

	// Room was just created: lastMessageAt = now. Should not be expired.
	if r.Expired(msgTTL, emptyTTL, time.Now()) {
		t.Error("new room should not be expired")
	}

	// Simulate 2 hours passing since last message.
	r.mu.Lock()
	r.lastMessageAt = time.Now().Add(-2*time.Hour - time.Second)
	r.mu.Unlock()

	if !r.Expired(msgTTL, emptyTTL, time.Now()) {
		t.Error("room with no messages for 2h should be expired")
	}
}

func TestRoomExpiredByEmptyRoom(t *testing.T) {
	m := NewManager()
	r := m.Create("test", "", "user1", 50, true)

	msgTTL := 2 * time.Hour
	emptyTTL := 15 * time.Minute

	// Room has users — lastUserLeftAt is zero. Should not expire via empty check.
	if r.Expired(msgTTL, emptyTTL, time.Now()) {
		t.Error("room with zero lastUserLeftAt should not expire via empty check")
	}

	// All users left 15+ minutes ago.
	r.TouchUserLeft()
	r.mu.Lock()
	r.lastUserLeftAt = time.Now().Add(-15*time.Minute - time.Second)
	// Keep lastMessageAt recent so only the empty check triggers.
	r.lastMessageAt = time.Now()
	r.mu.Unlock()

	if !r.Expired(msgTTL, emptyTTL, time.Now()) {
		t.Error("room empty for 15min should be expired")
	}
}

func TestRoomNotExpiredWhenUsersPresent(t *testing.T) {
	m := NewManager()
	r := m.Create("test", "", "user1", 50, true)

	msgTTL := 2 * time.Hour
	emptyTTL := 15 * time.Minute

	// Someone left, making room empty.
	r.TouchUserLeft()

	// But then someone joined, clearing the left timestamp.
	r.ClearUserLeft()

	// Even with recent lastMessageAt, the room should not expire.
	if r.Expired(msgTTL, emptyTTL, time.Now()) {
		t.Error("room should not be expired after user joined back")
	}
}

func TestRoomTouchMessageResetsExpiration(t *testing.T) {
	m := NewManager()
	r := m.Create("test", "", "user1", 50, true)

	msgTTL := 2 * time.Hour
	emptyTTL := 15 * time.Minute

	// Set last message to 2h ago.
	r.mu.Lock()
	r.lastMessageAt = time.Now().Add(-2*time.Hour - time.Second)
	r.mu.Unlock()

	if !r.Expired(msgTTL, emptyTTL, time.Now()) {
		t.Fatal("expected room to be expired before touch")
	}

	// Send a new message.
	r.TouchMessage()

	if r.Expired(msgTTL, emptyTTL, time.Now()) {
		t.Error("room should not be expired after TouchMessage")
	}
}

func TestManagerReapExpiresRooms(t *testing.T) {
	m := NewManager()
	r1 := m.Create("active", "", "user1", 50, true)
	r2 := m.Create("stale", "", "user1", 50, true)

	m.msgTTL = 2 * time.Hour
	m.emptyTTL = 15 * time.Minute

	// Make r2 stale (no messages for 2h+).
	r2.mu.Lock()
	r2.lastMessageAt = time.Now().Add(-3 * time.Hour)
	r2.mu.Unlock()

	var expiredIDs []string
	m.onExpire = func(roomID string) {
		expiredIDs = append(expiredIDs, roomID)
	}

	m.reap()

	if m.Get(r1.ID) == nil {
		t.Error("active room should not be deleted")
	}
	if m.Get(r2.ID) != nil {
		t.Error("stale room should be deleted")
	}
	if len(expiredIDs) != 1 || expiredIDs[0] != r2.ID {
		t.Errorf("expected onExpire for %q, got %v", r2.ID, expiredIDs)
	}
}

func TestManagerReapEmptyRoom(t *testing.T) {
	m := NewManager()
	r := m.Create("empty", "", "user1", 50, true)

	m.msgTTL = 2 * time.Hour
	m.emptyTTL = 15 * time.Minute

	// All users left 20 minutes ago.
	r.mu.Lock()
	r.lastUserLeftAt = time.Now().Add(-20 * time.Minute)
	r.mu.Unlock()

	var expired int32
	m.onExpire = func(roomID string) {
		atomic.AddInt32(&expired, 1)
	}

	m.reap()

	if m.Get(r.ID) != nil {
		t.Error("empty room should be deleted after 15min")
	}
	if atomic.LoadInt32(&expired) != 1 {
		t.Errorf("expected 1 expiration callback, got %d", atomic.LoadInt32(&expired))
	}
}

func TestManagerReapKeepsRecentEmptyRoom(t *testing.T) {
	m := NewManager()
	r := m.Create("just-emptied", "", "user1", 50, true)

	m.msgTTL = 2 * time.Hour
	m.emptyTTL = 15 * time.Minute

	// Users left just 5 minutes ago.
	r.mu.Lock()
	r.lastUserLeftAt = time.Now().Add(-5 * time.Minute)
	r.mu.Unlock()

	m.reap()

	if m.Get(r.ID) == nil {
		t.Error("recently-emptied room should not be deleted yet")
	}
}

func TestManagerStartExpirationReapsOverTime(t *testing.T) {
	m := NewManager()
	r := m.Create("stale", "", "user1", 50, true)
	roomID := r.ID

	// Set lastMessageAt far in the past so it's clearly expired.
	r.mu.Lock()
	r.lastMessageAt = time.Now().Add(-time.Hour)
	r.mu.Unlock()

	var expired atomic.Int32
	// Use 1ms TTLs — the room's lastMessageAt is 1h ago, well past 1ms.
	// The reap interval is max(ttl/2, 1s) = 1s, so we need to wait >1s.
	m.StartExpiration(time.Millisecond, time.Millisecond, func(roomID string) {
		expired.Add(1)
	})

	// Wait for at least one reap cycle (interval is 1s minimum).
	deadline := time.Now().Add(3 * time.Second)
	for m.Get(roomID) != nil && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}

	if m.Get(roomID) != nil {
		t.Error("room should have been reaped by the background loop")
	}
	if expired.Load() < 1 {
		t.Error("expected at least one expiration callback")
	}
}

func TestCreateRoomInitializesLastMessageAt(t *testing.T) {
	m := NewManager()
	before := time.Now()
	r := m.Create("test", "", "user1", 50, true)

	r.mu.Lock()
	lastMsg := r.lastMessageAt
	r.mu.Unlock()

	if lastMsg.Before(before) {
		t.Error("lastMessageAt should be set to creation time")
	}
}
