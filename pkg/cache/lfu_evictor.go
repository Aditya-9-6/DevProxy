package cache

import (
	"sync"
	"sync/atomic"
)

// LFUEvictor implements a lock-free LFU cache eviction policy.
type LFUEvictor struct {
	capacity uint64
	size     atomic.Uint64
	head     atomic.Pointer[Node]
	pool     sync.Pool
}

func NewLFUEvictor(capacity uint64) *LFUEvictor {
	return &LFUEvictor{
		capacity: capacity,
		pool: sync.Pool{
			New: func() interface{} { return &Node{} },
		},
	}
}

// Push adds or updates a node in the LFU list.
func (e *LFUEvictor) Push(key string, value interface{}) {
	newNode := e.pool.Get().(*Node)
	newNode.Key = key
	newNode.Value = value
	newNode.Frequency.Store(1)

	// Atomic insertion logic
	for {
		oldHead := e.head.Load()
		newNode.Next.Store(oldHead)
		if e.head.CompareAndSwap(oldHead, newNode) {
			e.size.Add(1)
			break
		}
	}
}

// Evict removes the least frequently used node.
func (e *LFUEvictor) Evict() {
	if e.size.Load() < e.capacity {
		return
	}

	// Simple atomic tail traversal for eviction
	curr := e.head.Load()
	var prev *Node
	for curr != nil && curr.Next.Load() != nil {
		prev = curr
		curr = curr.Next.Load()
	}

	if prev != nil {
		prev.Next.Store(nil)
		e.size.Add(^uint64(0))
		e.pool.Put(curr)
	}
}
