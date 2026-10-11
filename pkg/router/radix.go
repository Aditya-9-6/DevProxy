package router

import (
	"sync"
)

// Node represents a single segment in the radix tree.
type Node struct {
	children map[string]*Node
	handler  interface{}
	path     string
}

// RadixRouter provides thread-safe, high-performance path routing.
type RadixRouter struct {
	root *Node
	mu   sync.RWMutex
}

// NewRadixRouter initializes a new RadixRouter with a root node.
func NewRadixRouter() *RadixRouter {
	return &RadixRouter{root: &Node{children: make(map[string]*Node)}}
}

// Add registers a handler for a specific path. It handles multiple slashes and root paths.
func (r *RadixRouter) Add(path string, handler interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	curr := r.root
	start := -1
	for i := 0; i < len(path); i++ {
		if path[i] != '/' {
			if start == -1 {
				start = i
			}
		} else {
			if start != -1 {
				part := path[start:i]
				if curr.children == nil {
					curr.children = make(map[string]*Node)
				}
				if _, ok := curr.children[part]; !ok {
					curr.children[part] = &Node{children: make(map[string]*Node)}
				}
				curr = curr.children[part]
				start = -1
			}
		}
	}
	if start != -1 {
		part := path[start:]
		if curr.children == nil {
			curr.children = make(map[string]*Node)
		}
		if _, ok := curr.children[part]; !ok {
			curr.children[part] = &Node{children: make(map[string]*Node)}
		}
		curr = curr.children[part]
	}
	curr.handler = handler
	curr.path = path
}

// Lookup retrieves the handler for a given path. It is zero-allocation and handles multiple slashes.
func (r *RadixRouter) Lookup(path string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	curr := r.root
	start := -1
	for i := 0; i < len(path); i++ {
		if path[i] != '/' {
			if start == -1 {
				start = i
			}
		} else {
			if start != -1 {
				part := path[start:i]
				if curr.children == nil {
					return nil
				}
				if next, ok := curr.children[part]; ok {
					curr = next
				} else {
					return nil
				}
				start = -1
			}
		}
	}
	if start != -1 {
		part := path[start:]
		if curr.children == nil {
			return nil
		}
		if next, ok := curr.children[part]; ok {
			curr = next
		} else {
			return nil
		}
	}
	return curr.handler
}
