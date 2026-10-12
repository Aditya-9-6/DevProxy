package http2

import (
	"sync"
	"sync/atomic"
)

// Node represents a node in the HTTP/2 dependency tree.
type Node struct {
	StreamID uint32
	Weight   atomic.Uint32
	Parent   *Node
	Children map[uint32]*Node
}

// DependencyTree manages stream relationships with thread-safety.
// Uses a global RWMutex for consistency during structural modifications.
type DependencyTree struct {
	Root  *Node
	nodes map[uint32]*Node
	mu    sync.RWMutex
}

func NewDependencyTree() *DependencyTree {
	root := &Node{StreamID: 0, Children: make(map[uint32]*Node)}
	return &DependencyTree{
		Root:  root,
		nodes: map[uint32]*Node{0: root},
	}
}

func (t *DependencyTree) AddChild(parentID, childID uint32, weight uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()

	parent, ok := t.nodes[parentID]
	if !ok {
		parent = t.Root
	}

	curr := parent
	for curr != nil {
		if curr.StreamID == childID {
			return
		}
		curr = curr.Parent
	}

	node := &Node{StreamID: childID, Parent: parent, Children: make(map[uint32]*Node)}
	node.Weight.Store(weight)
	parent.Children[childID] = node
	t.nodes[childID] = node
}

func (t *DependencyTree) RemoveChild(streamID uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()

	node, ok := t.nodes[streamID]
	if !ok || node == t.Root {
		return
	}

	if node.Parent != nil {
		delete(node.Parent.Children, streamID)
	}
	delete(t.nodes, streamID)
}
