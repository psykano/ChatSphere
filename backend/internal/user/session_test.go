package user

import "testing"

func TestSessionStoreCreate(t *testing.T) {
	store := NewSessionStore()

	sess := store.Create()
	if sess.Token == "" {
		t.Fatal("expected non-empty token")
	}
	if sess.UserID == "" {
		t.Fatal("expected non-empty user ID")
	}
	if sess.CreatedAt.IsZero() {
		t.Fatal("expected non-zero created_at")
	}
}

func TestSessionStoreGet(t *testing.T) {
	store := NewSessionStore()

	sess := store.Create()
	got := store.Get(sess.Token)
	if got == nil {
		t.Fatal("expected to find session by token")
	}
	if got.UserID != sess.UserID {
		t.Errorf("expected user ID %q, got %q", sess.UserID, got.UserID)
	}
}

func TestSessionStoreGetNotFound(t *testing.T) {
	store := NewSessionStore()

	if got := store.Get("nonexistent"); got != nil {
		t.Errorf("expected nil for unknown token, got %+v", got)
	}
}

func TestSessionStoreCount(t *testing.T) {
	store := NewSessionStore()

	if store.Count() != 0 {
		t.Fatalf("expected 0 sessions, got %d", store.Count())
	}

	store.Create()
	store.Create()
	if store.Count() != 2 {
		t.Errorf("expected 2 sessions, got %d", store.Count())
	}
}

func TestSessionStoreUniqueTokens(t *testing.T) {
	store := NewSessionStore()

	s1 := store.Create()
	s2 := store.Create()
	if s1.Token == s2.Token {
		t.Error("expected unique tokens")
	}
	if s1.UserID == s2.UserID {
		t.Error("expected unique user IDs")
	}
}
