package h2priority

import (
	"sync"
	"sync/atomic"
)

// Node represents a cache-aligned HTTP/2 priority tree node with full parent-child tracking.
type Node struct {
	_        [56]byte // Padding for cache-line alignment
	Weight   atomic.Uint64
	StreamID uint32
	ParentID uint32
	Children map[uint32]*Node
	_        [40]byte // Padding
}

var nodePool = sync.Pool{
	New: func() interface{} {
		return &Node{
			Children: make(map[uint32]*Node),
		}
	},
}

func NewNode(streamID, parentID uint32, weight uint64) *Node {
	n := nodePool.Get().(*Node)
	n.StreamID = streamID
	n.ParentID = parentID
	n.Weight.Store(weight)
	if n.Children == nil {
		n.Children = make(map[uint32]*Node)
	}
	return n
}

func (n *Node) Release() {
	n.StreamID = 0
	n.ParentID = 0
	n.Weight.Store(0)
	for k := range n.Children {
		delete(n.Children, k)
	}
	nodePool.Put(n)
}
