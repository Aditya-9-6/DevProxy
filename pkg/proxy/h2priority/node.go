package h2priority

import (
	"sync"
	"sync/atomic"
)

// Node represents a cache-aligned HTTP/2 priority tree node.
type Node struct {
	_        [56]byte // Padding for cache-line alignment
	Weight   atomic.Uint64
	StreamID uint32
	ParentID uint32
	_        [48]byte // Padding
}

var nodePool = sync.Pool{
	New: func() interface{} {
		return &Node{}
	},
}

func NewNode(streamID, parentID uint32, weight uint64) *Node {
	n := nodePool.Get().(*Node)
	n.StreamID = streamID
	n.ParentID = parentID
	n.Weight.Store(weight)
	return n
}

func (n *Node) Release() {
	nodePool.Put(n)
}
