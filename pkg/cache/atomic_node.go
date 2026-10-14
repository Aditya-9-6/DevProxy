package cache

import (
	"sync/atomic"
)

// CacheLinePad prevents false sharing by padding structures to 64 bytes.
type CacheLinePad [8]uint64

// Node represents a cache entry node with frequency tracking.
type Node struct {
	Key       string
	Value     interface{}
	Frequency atomic.Uint64
	Next      atomic.Pointer[Node]
	Prev      atomic.Pointer[Node]
	_         CacheLinePad
}

// NewNode creates a new cache node.
func NewNode(key string, value interface{}) *Node {
	n := &Node{Key: key, Value: value}
	n.Frequency.Store(1)
	return n
}
