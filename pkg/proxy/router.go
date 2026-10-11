package proxy

import (
	"net/http"
	"sync"
)

type Router struct {
	mu        sync.RWMutex
	routes    map[string]string
	socketMap *SocketMap
}

func NewRouter() *Router {
	return &Router{
		routes:    make(map[string]string),
		socketMap: NewSocketMap(),
	}
}

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

func (r *Router) AddRoute(pattern, target string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[pattern] = target
	r.socketMap.Invalidate(pattern)
}
