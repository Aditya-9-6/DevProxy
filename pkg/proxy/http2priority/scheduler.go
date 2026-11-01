package http2priority

// Scheduler manages HTTP/2 priority tree traversal and scheduling.
type Scheduler struct {
	Root *Node
}

// NewScheduler creates a new Scheduler with a root node.
func NewScheduler() *Scheduler {
	root := AcquireNode(0, 16)
	return &Scheduler{
		Root: root,
	}
}

// Insert adds a stream node into the scheduler.
func (s *Scheduler) Insert(streamID uint32, parentID uint32, weight uint8, exclusive bool) {
	// Simple placeholder implementation for tree insertion
	child := AcquireNode(streamID, weight)
	AddChild(s.Root, child, weight, exclusive)
}
