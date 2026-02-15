package ratelimit

import (
	"sync"
	"time"
)

// IPLimiter tracks request counts per IP within a sliding window.
type IPLimiter struct {
	mu      sync.Mutex
	entries map[string][]time.Time
	max     int
	window  time.Duration
}

// NewIPLimiter creates an IPLimiter allowing max requests per window.
func NewIPLimiter(max int, window time.Duration) *IPLimiter {
	return &IPLimiter{
		entries: make(map[string][]time.Time),
		max:     max,
		window:  window,
	}
}

// Allow returns true if the IP has not exceeded the rate limit.
// If allowed, the request is recorded.
func (l *IPLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	timestamps := l.entries[ip]
	// Remove expired entries
	valid := timestamps[:0]
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= l.max {
		l.entries[ip] = valid
		return false
	}

	l.entries[ip] = append(valid, now)
	return true
}
