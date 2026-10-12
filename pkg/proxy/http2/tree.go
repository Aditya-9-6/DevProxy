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
	mu       sync.RWMutex
}

// DependencyTree manages stream relationships with thread-safety.
// Uses a flat map for O(1) lookups to avoid recursive traversal.
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

	// Cycle detection: ensure child is not an ancestor of parent
	curr := parent
	for curr != nil {
		if curr.StreamID == childID {
			return // Cycle detected, abort
		}
		curr = curr.Parent
	}

	node := &Node{StreamID: childID, Parent: parent, Children: make(map[uint32]*Node)}
	node.Weight.Store(weight)
	parent.Children[childID] = node
	t.nodes[childID] = node
}
