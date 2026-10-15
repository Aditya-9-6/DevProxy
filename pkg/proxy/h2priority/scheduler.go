package h2priority

import (
	"sync"
)

// Scheduler manages the dependency graph using a lock-free approach.
type Scheduler struct {
	mu    sync.RWMutex
	nodes map[uint32]*Node
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		nodes: make(map[uint32]*Node),
	}
}

func (s *Scheduler) UpdateWeight(streamID uint32, weight uint64) {
	s.mu.RLock()
	n, ok := s.nodes[streamID]
	s.mu.RUnlock()
	if ok {
		n.Weight.Store(weight)
	}
}

func (s *Scheduler) AddNode(streamID, parentID uint32, weight uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[streamID] = NewNode(streamID, parentID, weight)
}

func (s *Scheduler) RemoveNode(streamID uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n, ok := s.nodes[streamID]; ok {
		delete(s.nodes, streamID)
		n.Release()
	}
}
