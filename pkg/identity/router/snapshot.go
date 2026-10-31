package router

import (
	"sync/atomic"
	"github.com/Aditya-9-6/DevProxy/pkg/identity/spiffe"
)

type SnapshotRouter struct {
	root atomic.Value
}

func NewSnapshotRouter(root *Node) *SnapshotRouter {
	s := &SnapshotRouter{}
	s.root.Store(root)
	return s
}

func (s *SnapshotRouter) Route(id spiffe.ID) bool {
	rootVal := s.root.Load()
	if rootVal == nil {
		return false
	}
	rootNode, ok := rootVal.(*Node)
	if !ok {
		return false
	}
	return rootNode.Match(id)
}

func (s *SnapshotRouter) Update(root *Node) {
	s.root.Store(root)
}
