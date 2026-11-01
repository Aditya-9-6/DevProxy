package http2priority

import (
	"sync"
)

// Node represents an HTTP/2 priority stream node.
type Node struct {
	StreamID uint32
	Parent   *Node
	Children []*Node
	Weight   uint8
	Exclusive bool
}

var nodePool = sync.Pool{
	New: func() interface{} {
		return &Node{
			Children: make([]*Node, 0, 4),
		}
	},
}

// AcquireNode gets a Node from the pool.
func AcquireNode(streamID uint32, weight uint8) *Node {
	n := nodePool.Get().(*Node)
	n.StreamID = streamID
	n.Weight = weight
	n.Parent = nil
	n.Children = n.Children[:0]
	n.Exclusive = false
	return n
}

// ReleaseNode returns a Node to the pool.
func ReleaseNode(n *Node) {
	if n == nil {
		return
	}
	n.Parent = nil
	for _, child := range n.Children {
		ReleaseNode(child)
	}
	n.Children = n.Children[:0]
	nodePool.Put(n)
}

// AddChild adds a child node under a parent, respecting exclusive flag.
func AddChild(parent, child *Node, weight uint8, exclusive bool) {
	if parent == nil || child == nil {
		return
	}
	child.Weight = weight
	child.Parent = parent
	child.Exclusive = exclusive

	if exclusive {
		for _, existing := range parent.Children {
			existing.Parent = child
			child.Children = append(child.Children, existing)
		}
		parent.Children = parent.Children[:0]
	}

	parent.Children = append(parent.Children, child)
}

// RemoveNode removes a node from its parent.
func RemoveNode(n *Node) {
	if n == nil || n.Parent == nil {
		return
	}
	p := n.Parent
	idx := -1
	for i, c := range p.Children {
		if c.StreamID == n.StreamID {
			idx = i
			break
		}
	}
	if idx >= 0 {
		p.Children = append(p.Children[:idx], p.Children[idx+1:]...)
	}
	for _, c := range n.Children {
		c.Parent = p
		p.Children = append(p.Children, c)
	}
	n.Children = n.Children[:0]
	n.Parent = nil
}
