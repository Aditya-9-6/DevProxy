package router

import (
	"strings"
	"github.com/Aditya-9-6/DevProxy/pkg/identity/spiffe"
)

type Node struct {
	children map[string]*Node
	values   []spiffe.ID
}

func NewNode() *Node {
	return &Node{
		children: make(map[string]*Node),
	}
}

func (n *Node) Insert(id spiffe.ID) {
	parts := strings.Split(id.Path, "/")
	curr := n
	for _, part := range parts {
		if part == "" {
			continue
		}
		if curr.children == nil {
			curr.children = make(map[string]*Node)
		}
		child, ok := curr.children[part]
		if !ok {
			child = NewNode()
			curr.children[part] = child
		}
		curr = child
	}
	curr.values = append(curr.values, id)
}

func (n *Node) Match(id spiffe.ID) bool {
	parts := strings.Split(id.Path, "/")
	curr := n
	for _, part := range parts {
		if part == "" {
			continue
		}
		if curr.children == nil {
			return false
		}
		child, ok := curr.children[part]
		if !ok {
			return false
		}
		curr = child
	}
	return len(curr.values) > 0
}
