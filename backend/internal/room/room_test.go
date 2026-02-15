package room

import (
	"testing"
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
