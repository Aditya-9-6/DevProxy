package proxy

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"sync/atomic"
)

const SocketMapSize = 1024

var entryPool = sync.Pool{
	New: func() interface{} {
		return &SocketEntry{}
	},
}

type SocketEntry struct {
	TargetAddr string
	IsActive   atomic.Bool
}

type SocketMap struct {
	buckets [SocketMapSize]atomic.Pointer[SocketEntry]
	seed    uint32
}

func NewSocketMap() *SocketMap {
	var b [4]byte
	rand.Read(b[:])
	return &SocketMap{
		seed: binary.LittleEndian.Uint32(b[:]),
	}
}

func (sm *SocketMap) hash(key string) uint32 {
	hash := uint32(2166136261) ^ sm.seed
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return hash % SocketMapSize
}

func (sm *SocketMap) Get(key string) string {
	idx := sm.hash(key)
	entry := sm.buckets[idx].Load()
	if entry != nil && entry.IsActive.Load() {
		return entry.TargetAddr
	}
	return ""
}

func (sm *SocketMap) Set(key, target string) {
	idx := sm.hash(key)
	entry := entryPool.Get().(*SocketEntry)
	entry.TargetAddr = target
	entry.IsActive.Store(true)
	sm.buckets[idx].Store(entry)
}

func (sm *SocketMap) Invalidate(key string) {
	idx := sm.hash(key)
	entry := sm.buckets[idx].Load()
	if entry != nil {
		entry.IsActive.Store(false)
	}
}
