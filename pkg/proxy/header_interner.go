package proxy

import (
	"sync"
)

const (
	InternerSize = 1024
)

type bucket struct {
	sync.RWMutex
	entries map[string]string
	_       [56]byte // Padding to prevent false sharing (cache-line alignment)
}

type HeaderInterner struct {
	buckets [InternerSize]*bucket
}

func NewHeaderInterner() *HeaderInterner {
	h := &HeaderInterner{}
	for i := 0; i < InternerSize; i++ {
		h.buckets[i] = &bucket{entries: make(map[string]string)}
	}
	return h
}

// hashFNV1a implements a stack-based FNV-1a hash to avoid heap allocations.
func hashFNV1a(s string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		hash ^= uint32(s[i])
		hash *= 16777619
	}
	return hash
}

func (h *HeaderInterner) Intern(key string) string {
	idx := int(hashFNV1a(key) % InternerSize)
	b := h.buckets[idx]

	b.RLock()
	if val, ok := b.entries[key]; ok {
		b.RUnlock()
		return val
	}
	b.RUnlock()

	b.Lock()
	defer b.Unlock()

	if val, ok := b.entries[key]; ok {
		return val
	}

	b.entries[key] = key
	return key
}
