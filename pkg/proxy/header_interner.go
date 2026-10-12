package proxy

import (
	"hash/fnv"
	"sync"
)

const (
	InternerSize = 1024
)

type headerEntry struct {
	key string
}

type bucket struct {
	sync.RWMutex
	entries map[string]string
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

func (h *HeaderInterner) Intern(key string) string {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	idx := int(hash.Sum32() % InternerSize)

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
