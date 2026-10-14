package proxy

import (
	"sync/atomic"
	"unsafe"
)

const (
	DictSize  = 1024
	CacheLine = 64
)

type dictEntry struct {
	key   atomic.Pointer[string]
	value atomic.Pointer[string]
	_     [CacheLine - unsafe.Sizeof(atomic.Pointer[string]{})*2]byte
}

type AtomicDictionary struct {
	entries [DictSize]dictEntry
	cursor  atomic.Uint64
}

func NewAtomicDictionary() *AtomicDictionary {
	return &AtomicDictionary{}
}

func (d *AtomicDictionary) Put(key, value string) {
	idx := d.cursor.Add(1) % DictSize
	d.entries[idx].key.Store(&key)
	d.entries[idx].value.Store(&value)
}

func (d *AtomicDictionary) Get(key string) (string, bool) {
	for i := 0; i < DictSize; i++ {
		kPtr := d.entries[i].key.Load()
		if kPtr != nil && *kPtr == key {
			vPtr := d.entries[i].value.Load()
			if vPtr != nil {
				return *vPtr, true
			}
		}
	}
	return "", false
}
