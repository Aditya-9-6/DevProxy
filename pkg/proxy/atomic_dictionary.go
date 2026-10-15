package proxy

import (
	"sync/atomic"
	"unsafe"
)

const (
	DictSize  = 1024 // Must be a power of two for bitwise masking
	CacheLine = 64
)

type entryPayload struct {
	key   string
	value string
}

type dictEntry struct {
	ptr atomic.Pointer[entryPayload]
	_   [CacheLine - unsafe.Sizeof(atomic.Pointer[entryPayload]{})]byte
}

type AtomicDictionary struct {
	entries [DictSize]dictEntry
}

func NewAtomicDictionary() *AtomicDictionary {
	return &AtomicDictionary{}
}

func hashKey(key string) uint32 {
	var hash uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return hash
}

func (d *AtomicDictionary) Put(key, value string) {
	hash := hashKey(key)
	startIdx := hash & (DictSize - 1)

	for i := 0; i < 8; i++ {
		idx := (startIdx + uint32(i)) & (DictSize - 1)
		entry := &d.entries[idx]
		curr := entry.ptr.Load()
		if curr == nil || curr.key == key {
			newPayload := &entryPayload{key: key, value: value}
			if entry.ptr.CompareAndSwap(curr, newPayload) {
				return
			}
		}
	}

	// Fallback linear probe if collision group is full
	for i := uint32(0); i < DictSize; i++ {
		idx := (startIdx + i) & (DictSize - 1)
		entry := &d.entries[idx]
		curr := entry.ptr.Load()
		if curr == nil || curr.key == key {
			newPayload := &entryPayload{key: key, value: value}
			if entry.ptr.CompareAndSwap(curr, newPayload) {
				return
			}
		}
	}
}

func (d *AtomicDictionary) Get(key string) (string, bool) {
	hash := hashKey(key)
	startIdx := hash & (DictSize - 1)

	for i := 0; i < 8; i++ {
		idx := (startIdx + uint32(i)) & (DictSize - 1)
		curr := d.entries[idx].ptr.Load()
		if curr != nil && curr.key == key {
			return curr.value, true
		}
	}

	// Fallback linear probe
	for i := uint32(0); i < DictSize; i++ {
		idx := (startIdx + i) & (DictSize - 1)
		curr := d.entries[idx].ptr.Load()
		if curr == nil {
			break
		}
		if curr.key == key {
			return curr.value, true
		}
	}

	return "", false
}
