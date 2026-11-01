package quicretry

import (
	"sync"
	"time"
)

// ReplayCache tracks recently used nonces or tokens to prevent replay attacks.
type ReplayCache struct {
	mu      sync.Mutex
	entries map[string]time.Time
	ttl     time.Duration
}

// NewReplayCache creates a replay cache with the specified TTL.
func NewReplayCache(ttl time.Duration) *ReplayCache {
	rc := &ReplayCache{
		entries: make(map[string]time.Time),
		ttl:     ttl,
	}
	go rc.cleanupLoop(1 * time.Minute)
	return rc
}

// CheckAndAdd checks if a token/nonce exists. If not, it adds it and returns true.
// If it already exists, it returns false (replay detected).
func (rc *ReplayCache) CheckAndAdd(identifier string) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	if t, exists := rc.entries[identifier]; exists {
		if now.Before(t) {
			return false
		}
	}

	rc.entries[identifier] = now.Add(rc.ttl)
	return true
}

func (rc *ReplayCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for k, exp := range rc.entries {
			if now.After(exp) {
				delete(rc.entries, k)
			}
		}
		rc.mu.Unlock()
	}
}
