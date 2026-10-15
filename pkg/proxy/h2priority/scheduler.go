package h2priority

import (
	"sync"
)

// Scheduler manages the HTTP/2 priority dependency graph with proper thread safety and hierarchy.
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if n, ok := s.nodes[streamID]; ok {
		n.Weight.Store(weight)
	}
}

func (s *Scheduler) AddNode(streamID, parentID uint32, weight uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[streamID]; exists {
		s.removeNodeLocked(streamID)
	}

	n := NewNode(streamID, parentID, weight)
	s.nodes[streamID] = n

	if parentNode, ok := s.nodes[parentID]; ok {
		parentNode.Children[streamID] = n
	} else if parentID != 0 {
		// If parent does not exist, attach to root (0)
		n.ParentID = 0
	}
}

func (s *Scheduler) RemoveNode(streamID uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeNodeLocked(streamID)
}

func (s *Scheduler) removeNodeLocked(streamID uint32) {
	n, ok := s.nodes[streamID]
	if !ok {
		return
	}

	// Reparent children to root or parent
	if parentNode, ok := s.nodes[n.ParentID]; ok {
		delete(parentNode.Children, streamID)
		for _, child := range n.Children {
			child.ParentID = parentNode.StreamID
			parentNode.Children[child.StreamID] = child
		}
	} else {
		for _, child := range n.Children {
			child.ParentID = 0
		}
	}

	delete(s.nodes, streamID)
	n.Release()
}
