package http2

import "sync/atomic"

// Node represents a node in the HTTP/2 dependency tree.
type Node struct {
	StreamID uint32
	Weight   atomic.Uint32
	Parent   *Node
}

// DependencyTree manages stream relationships.
type DependencyTree struct {
	Root *Node
}

func NewDependencyTree() *DependencyTree {
	return &DependencyTree{
		Root: &Node{StreamID: 0},
	}
}
