package router

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/Aditya-9-6/DevProxy/pkg/identity/spiffe"
)

// ErrNilPolicy indicates that a nil policy cannot be published.
var ErrNilPolicy = errors.New("router: policy is nil")

// Snapshot is an immutable routing view. Its policy and radix trees are never
// mutated after publication, allowing readers to retain a snapshot safely.
type Snapshot struct {
	policy *spiffe.Policy
	roots  map[string]*pathTrie
}

// Router publishes immutable route snapshots with atomic pointer swaps.
type Router struct {
	current atomic.Pointer[Snapshot]
}

// NewRouter builds and atomically publishes the initial routing snapshot.
func NewRouter(policy *spiffe.Policy) (*Router, error) {
	snapshot, err := newSnapshot(policy)
	if err != nil {
		return nil, err
	}
	router := &Router{}
	router.current.Store(snapshot)
	return router, nil
}

// Publish builds a complete replacement snapshot and atomically swaps it in.
// A failed build leaves the currently published snapshot untouched.
func (r *Router) Publish(policy *spiffe.Policy) error {
	if r == nil {
		return fmt.Errorf("router: router is nil")
	}
	snapshot, err := newSnapshot(policy)
	if err != nil {
		return err
	}
	r.current.Store(snapshot)
	return nil
}

// Current returns the currently published immutable snapshot.
func (r *Router) Current() *Snapshot {
	if r == nil {
		return nil
	}
	return r.current.Load()
}

// Lookup resolves an identity against one immutable snapshot. Successful
// lookups perform no heap allocations.
func (r *Router) Lookup(id spiffe.ID) (Route, bool) {
	snapshot := r.Current()
	if snapshot == nil {
		return Route{}, false
	}
	return snapshot.Lookup(id)
}

// Lookup resolves an identity within this immutable snapshot.
func (s *Snapshot) Lookup(id spiffe.ID) (Route, bool) {
	if s == nil || s.policy == nil || !s.policy.Allows(id) {
		return Route{}, false
	}
	root := s.roots[id.TrustDomain]
	if root == nil {
		return Route{}, false
	}
	path := id.Path
	if path == "" {
		path = "/"
	}
	return root.lookup(path)
}

// Policy returns the immutable policy associated with the snapshot.
func (s *Snapshot) Policy() *spiffe.Policy {
	if s == nil {
		return nil
	}
	return s.policy
}

func newSnapshot(policy *spiffe.Policy) (*Snapshot, error) {
	if policy == nil {
		return nil, ErrNilPolicy
	}

	routes := policy.Routes()
	roots := make(map[string]*pathTrie, len(routes))
	for _, route := range routes {
		root := roots[route.TrustDomain]
		if root == nil {
			root = newPathTrie()
			roots[route.TrustDomain] = root
		}
		if err := root.insert(route); err != nil {
			return nil, fmt.Errorf("router: build snapshot: %w", err)
		}
	}
	return &Snapshot{policy: policy, roots: roots}, nil
}
