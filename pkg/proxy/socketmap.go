package proxy

import (
	"sync/atomic"
)

const SocketMapSize = 1024

type SocketEntry struct {
	TargetAddr string
	IsActive   atomic.Bool
}

type SocketMap struct {
	buckets [SocketMapSize]*SocketEntry
}

func NewSocketMap() *SocketMap {
	return &SocketMap{}
}

func (sm *SocketMap) hash(key string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return hash % SocketMapSize
}

func (sm *SocketMap) Get(key string) string {
	idx := sm.hash(key)
	entry := sm.buckets[idx]
	if entry != nil && entry.IsActive.Load() {
		return entry.TargetAddr
	}
	return ""
}

func (sm *SocketMap) Set(key, target string) {
	idx := sm.hash(key)
	sm.buckets[idx] = &SocketEntry{TargetAddr: target}
	sm.buckets[idx].IsActive.Store(true)
}

func (sm *SocketMap) Invalidate(key string) {
	idx := sm.hash(key)
	if sm.buckets[idx] != nil {
		sm.buckets[idx].IsActive.Store(false)
	}
}
