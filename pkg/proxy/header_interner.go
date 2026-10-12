package proxy

import (
	"hash/fnv"
	"sync/atomic"
)

const (
	InternerSize = 1024
	CacheLinePad = 64
)

type headerEntry struct {
	key   string
	value string
}

type HeaderInterner struct {
	_       [CacheLinePad]byte
	buckets atomic.Pointer[[]headerEntry]
	_       [CacheLinePad]byte
}

func NewHeaderInterner() *HeaderInterner {
	initial := make([]headerEntry, InternerSize)
	h := &HeaderInterner{}
	h.buckets.Store(&initial)
	return h
}

func (h *HeaderInterner) Intern(key string) string {
	hash := fnv.New32a()
	hash.Write([]byte(key))
	idx := int(hash.Sum32() % InternerSize)

	buckets := *h.buckets.Load()
	entry := &buckets[idx]

	if entry.key == key {
		return entry.key
	}

	entry.key = key
	return key
}
