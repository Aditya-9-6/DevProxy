package cache

import (
	"sync"
)

// LFUEvictor implements a thread-safe LFU cache eviction policy.
type LFUEvictor struct {
	mu       sync.Mutex
	capacity uint64
	size     uint64
	head     *Node
	tail     *Node
}

func NewLFUEvictor(capacity uint64) *LFUEvictor {
	return &LFUEvictor{capacity: capacity}
}

// Push adds or updates a node in the LFU list.
func (e *LFUEvictor) Push(key string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	node := &Node{Key: key, Value: value, Frequency: 1}
	node.Next = e.head
	if e.head != nil {
		e.head.Prev = node
	} else {
		e.tail = node
	}
	e.head = node
	e.size++
}

// Evict removes the least frequently used node.
func (e *LFUEvictor) Evict() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.size == 0 || e.size < e.capacity {
		return
	}

	// Find LFU node (simple scan for demonstration of LFU logic)
	var lfu *Node = e.tail
	curr := e.tail
	for curr != nil {
		if curr.Frequency < lfu.Frequency {
			lfu = curr
		}
		curr = curr.Prev
	}

	// Remove lfu node
	if lfu.Prev != nil {
		lfu.Prev.Next = lfu.Next
	} else {
		e.head = lfu.Next
	}
	if lfu.Next != nil {
		lfu.Next.Prev = lfu.Prev
	} else {
		e.tail = lfu.Prev
	}
	e.size--
}
