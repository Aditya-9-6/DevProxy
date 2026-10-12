package http2

import (
	"sync"
	"sync/atomic"
)

// Node represents a node in the HTTP/2 dependency tree.
// Traversal complexity is O(depth), where depth is typically small in HTTP/2.
type Node struct {
	StreamID uint32
	Weight   atomic.Uint32
	Parent   *Node
	Children map[uint32]*Node
	mu       sync.RWMutex
}

// DependencyTree manages stream relationships with thread-safety.
type DependencyTree struct {
	Root *Node
	mu   sync.RWMutex
}

func NewDependencyTree() *DependencyTree {
	return &DependencyTree{
		Root: &Node{StreamID: 0, Children: make(map[uint32]*Node)},
	}
}

func (t *DependencyTree) AddChild(parentID, childID uint32, weight uint32) {
	t.mu.Lock()
	defer t.mu.Unlock()

	parent := t.findNode(t.Root, parentID)
	if parent == nil {
		parent = t.Root
	}

	parent.mu.Lock()
	defer parent.mu.Unlock()

	node := &Node{StreamID: childID, Parent: parent, Children: make(map[uint32]*Node)}
	node.Weight.Store(weight)
	parent.Children[childID] = node
}

func (t *DependencyTree) findNode(curr *Node, id uint32) *Node {
	if curr.StreamID == id {
		return curr
	}
	curr.mu.RLock()
	defer curr.mu.RUnlock()
	for _, child := range curr.Children {
		if found := t.findNode(child, id); found != nil {
			return found
		}
	}
	return nil
}
