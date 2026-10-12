package replay

import (
	"sync"
	"sync/atomic"
)

// StrikeRegister implements RFC 8446 0-RTT anti-replay.
type StrikeRegister struct {
	currentFilter *AtomicBloomFilter
	nextFilter    *AtomicBloomFilter
	epoch         atomic.Uint64
	mu            sync.RWMutex
}

func NewStrikeRegister(size uint64) *StrikeRegister {
	return &StrikeRegister{
		currentFilter: NewAtomicBloomFilter(size),
		nextFilter:    NewAtomicBloomFilter(size),
	}
}

func (s *StrikeRegister) IsReplay(digest uint64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.currentFilter.Test(digest) {
		return true
	}
	s.currentFilter.Add(digest)
	return false
}

func (s *StrikeRegister) Rotate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentFilter = s.nextFilter
	s.nextFilter = NewAtomicBloomFilter(s.currentFilter.size)
	s.epoch.Add(1)
}
