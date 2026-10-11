package proxy

import (
	"net/http"
	"sync"

	"github.com/Aditya-9-6/DevProxy/pkg/router"
)

// Router provides high-performance routing for the proxy using a socket map for fast-path lookups.
type Router struct {
	mu        sync.RWMutex
	routes    map[string]string
	socketMap *SocketMap
}

// NewRouter initializes a new Router with an empty route map and a fresh socket map.
func NewRouter() *Router {
	return &Router{
		routes:    make(map[string]string),
		socketMap: NewSocketMap(),
	}
}

// Route determines the target URL for a given request.
func (r *Router) Route(req *http.Request) string {
	// 1. Fast Path: Lock-free socket map lookup
	key := req.Host + req.URL.Path
	if target := r.socketMap.Get(key); target != "" {
		return target
	}

	// 2. Slow Path: Standard routing logic
	r.mu.RLock()
	target, ok := r.routes[key]
	r.mu.RUnlock()

	if ok {
		r.socketMap.Set(key, target)
		return target
	}

	return ""
}

// AddRoute registers a new pattern and its corresponding target URL.
func (r *Router) AddRoute(pattern, target string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[pattern] = target
	r.socketMap.Invalidate(pattern)
}

// ProxyRouter is a radix-tree based router for internal proxy dispatching.
type ProxyRouter struct {
	engine *router.RadixRouter
}

// NewProxyRouter creates a new ProxyRouter instance.
func NewProxyRouter() *ProxyRouter {
	return &ProxyRouter{engine: router.NewRadixRouter()}
}

// Register adds a path and its associated handler to the radix tree.
func (pr *ProxyRouter) Register(path string, handler interface{}) {
	pr.engine.Add(path, handler)
}

// Route performs a lookup in the radix tree for the given path.
func (pr *ProxyRouter) Route(path string) interface{} {
	return pr.engine.Lookup(path)
}
