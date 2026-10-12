package tls

import (
	"sync/atomic"
)

// ProtocolRouter provides lock-free routing of ALPN protocols to handlers.
type ProtocolRouter struct {
	routes atomic.Pointer[map[string]ProtocolHandler]
}

func NewProtocolRouter() *ProtocolRouter {
	r := &ProtocolRouter{}
	r.routes.Store(&map[string]ProtocolHandler{})
	return r
}

func (r *ProtocolRouter) Register(proto string, handler ProtocolHandler) {
	for {
		old := r.routes.Load()
		newRoutes := make(map[string]ProtocolHandler)
		for k, v := range *old {
			newRoutes[k] = v
		}
		newRoutes[proto] = handler
		if r.routes.CompareAndSwap(old, &newRoutes) {
			break
		}
	}
}

func (r *ProtocolRouter) Route(proto string) (ProtocolHandler, bool) {
	routes := *r.routes.Load()
	handler, ok := routes[proto]
	return handler, ok
}

func (r *ProtocolRouter) SupportedProtocols() []string {
	routes := *r.routes.Load()
	protos := make([]string, 0, len(routes))
	for k := range routes {
		protos = append(protos, k)
	}
	return protos
}
