package cache

import (
	"sync"
)

// LFUEvictor implements a thread-safe O(1) LFU cache eviction policy.
type LFUEvictor struct {
	mu       sync.Mutex
	capacity uint64
	size     uint64
	nodes    map[string]*Node
	freqs    map[uint64]*FreqList
	minFreq  uint64
	pool     sync.Pool
}

func NewLFUEvictor(capacity uint64) *LFUEvictor {
	return &LFUEvictor{
		capacity: capacity,
		nodes:    make(map[string]*Node),
		freqs:    make(map[uint64]*FreqList),
		pool:     sync.Pool{New: func() interface{} { return &Node{} }},
	}
}

func (e *LFUEvictor) Push(key string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if node, ok := e.nodes[key]; ok {
		node.Value = value
		e.increment(node)
		return
	}

	if e.size >= e.capacity {
		e.evictInternal()
	}

	node := e.pool.Get().(*Node)
	node.Key, node.Value, node.Frequency = key, value, 1
	e.nodes[key] = node
	e.addToFreqList(1, node)
	e.minFreq = 1
	e.size++
}

func (e *LFUEvictor) increment(node *Node) {
	e.removeFromFreqList(node)
	node.Frequency++
	e.addToFreqList(node.Frequency, node)
	if e.minFreq == node.Frequency-1 && e.freqs[node.Frequency-1] == nil {
		e.minFreq = node.Frequency
	}
}

func (e *LFUEvictor) addToFreqList(f uint64, node *Node) {
	if e.freqs[f] == nil {
		e.freqs[f] = &FreqList{}
	}
	list := e.freqs[f]
	node.List = list
	node.Next = list.Head
	node.Prev = nil
	if list.Head != nil {
		list.Head.Prev = node
	}
	list.Head = node
	if list.Tail == nil {
		list.Tail = node
	}
}

func (e *LFUEvictor) removeFromFreqList(node *Node) {
	list := node.List
	if node.Prev != nil {
		node.Prev.Next = node.Next
	} else {
		list.Head = node.Next
	}
	if node.Next != nil {
		node.Next.Prev = node.Prev
	} else {
		list.Tail = node.Prev
	}
	if list.Head == nil {
		delete(e.freqs, node.Frequency)
	}
}

func (e *LFUEvictor) Evict() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.evictInternal()
}

func (e *LFUEvictor) evictInternal() {
	if e.size == 0 {
		return
	}
	list := e.freqs[e.minFreq]
	if list == nil || list.Tail == nil {
		return
	}
	node := list.Tail
	e.removeFromFreqList(node)
	delete(e.nodes, node.Key)
	e.size--
	e.pool.Put(node)
}
