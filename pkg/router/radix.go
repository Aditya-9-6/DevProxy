package router

import (
	"strings"
	"sync"
)

type Node struct {
	children map[string]*Node
	handler  interface{}
	path     string
}

type RadixRouter struct {
	root *Node
	mu   sync.RWMutex
}

func NewRadixRouter() *RadixRouter {
	return &RadixRouter{root: &Node{children: make(map[string]*Node)}}
}

func (r *RadixRouter) Add(path string, handler interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	parts := strings.Split(strings.Trim(path, "/"), "/")
	curr := r.root
	for _, part := range parts {
		if _, ok := curr.children[part]; !ok {
			curr.children[part] = &Node{children: make(map[string]*Node)}
		}
		curr = curr.children[part]
	}
	curr.handler = handler
	curr.path = path
}

func (r *RadixRouter) Lookup(path string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	parts := strings.Split(strings.Trim(path, "/"), "/")
	curr := r.root
	for _, part := range parts {
		if next, ok := curr.children[part]; ok {
			curr = next
		} else {
			return nil
		}
	}
	return curr.handler
}
