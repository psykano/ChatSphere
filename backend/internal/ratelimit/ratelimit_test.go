package ratelimit

import (
	"testing"
	"time"
)

func TestAllowUnderLimit(t *testing.T) {
	l := NewIPLimiter(3, time.Hour)

	for i := 0; i < 3; i++ {
		if !l.Allow("1.2.3.4") {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
}

func TestDenyOverLimit(t *testing.T) {
	l := NewIPLimiter(3, time.Hour)

	for i := 0; i < 3; i++ {
		l.Allow("1.2.3.4")
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("4th request should be denied")
	}
}

func TestDifferentIPsIndependent(t *testing.T) {
	l := NewIPLimiter(2, time.Hour)

	l.Allow("1.1.1.1")
	l.Allow("1.1.1.1")

	if l.Allow("1.1.1.1") {
		t.Fatal("1.1.1.1 should be denied")
	}
	if !l.Allow("2.2.2.2") {
		t.Fatal("2.2.2.2 should be allowed")
	}
}

func TestExpiredEntriesPruned(t *testing.T) {
	l := NewIPLimiter(2, 50*time.Millisecond)

	l.Allow("1.2.3.4")
	l.Allow("1.2.3.4")

	if l.Allow("1.2.3.4") {
		t.Fatal("should be denied before window expires")
	}

	time.Sleep(60 * time.Millisecond)

	if !l.Allow("1.2.3.4") {
		t.Fatal("should be allowed after window expires")
	}
}
