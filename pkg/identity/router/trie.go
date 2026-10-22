package router

import (
	"errors"
	"fmt"

	"github.com/Aditya-9-6/DevProxy/pkg/identity/spiffe"
)

// Route is the immutable route definition returned by snapshot lookups.
type Route = spiffe.Route

var errDuplicateRoute = errors.New("router: duplicate route")

type trieNode struct {
	children map[byte]*trieNode
	route    *Route
}

type pathTrie struct {
	root *trieNode
}

func newPathTrie() *pathTrie {
	return &pathTrie{root: &trieNode{}}
}

func (t *pathTrie) insert(route Route) error {
	if route.Path == "" {
		return fmt.Errorf("router: route path is empty")
	}
	if route.Path == "/" {
		return t.insertRoot(route)
	}

	node := t.root
	for i := 0; i < len(route.Path); i++ {
		child := node.children[route.Path[i]]
		if child == nil {
			child = &trieNode{}
			if node.children == nil {
				node.children = make(map[byte]*trieNode)
			}
			node.children[route.Path[i]] = child
		}
		node = child
	}
	if node.route != nil {
		return fmt.Errorf("%w: %s", errDuplicateRoute, route.Path)
	}
	copied := route
	node.route = &copied
	return nil
}

func (t *pathTrie) insertRoot(route Route) error {
	if t.root.route != nil {
		return fmt.Errorf("%w: %s", errDuplicateRoute, route.Path)
	}
	copied := route
	t.root.route = &copied
	return nil
}

func (t *pathTrie) lookup(path string) (Route, bool) {
	if t == nil || t.root == nil {
		return Route{}, false
	}

	node := t.root
	best := node.route
	for i := 0; i < len(path); i++ {
		child := node.children[path[i]]
		if child == nil {
			break
		}
		node = child
		if node.route != nil && pathBoundary(path, i) {
			best = node.route
		}
	}
	if best == nil {
		return Route{}, false
	}
	return *best, true
}

func pathBoundary(path string, routeEnd int) bool {
	return routeEnd+1 == len(path) || path[routeEnd+1] == '/'
}
