package proxy

import (
	"net/http"
	"net/netip"

	"github.com/Aditya-9-6/DevProxy/pkg/waf"
)

// DefaultWAFIPCardinalityThreshold is a conservative starting threshold for
// deployments observing unusually high client-address diversity.
const DefaultWAFIPCardinalityThreshold uint64 = 10_000

// WAFIPCardinalityOptions configures client-IP cardinality inspection.
type WAFIPCardinalityOptions struct {
	Threshold uint64

	// ResolveClientIP may be supplied by deployments that trust a specific
	// proxy header. By default only RemoteAddr is used.
	ResolveClientIP func(*http.Request) netip.Addr

	// OnAnomaly is invoked when the estimated distinct-client count reaches
	// Threshold. It can attach a WAF finding to the request context.
	OnAnomaly func(*http.Request, uint64)

	// RejectAnomalies returns HTTP 429 when the threshold is reached.
	RejectAnomalies bool
}

// WAFIPCardinalityMiddleware tracks distinct upstream client addresses using
// a lock-free HyperLogLog without buffering request bodies.
type WAFIPCardinalityMiddleware struct {
	tracker *waf.IPCardinalityTracker
	options WAFIPCardinalityOptions
}

// NewWAFIPCardinalityMiddleware constructs a cardinality inspection middleware.
func NewWAFIPCardinalityMiddleware(options WAFIPCardinalityOptions) *WAFIPCardinalityMiddleware {
	return &WAFIPCardinalityMiddleware{
		tracker: waf.NewIPCardinalityTracker(options.Threshold),
		options: options,
	}
}

// NewWAFIPCardinalityMiddlewareForThreshold constructs a middleware using the
// default resolver and anomaly callback behavior.
func NewWAFIPCardinalityMiddlewareForThreshold(threshold uint64) *WAFIPCardinalityMiddleware {
	return NewWAFIPCardinalityMiddleware(WAFIPCardinalityOptions{Threshold: threshold})
}

// ObserveRequest records the request's client address and reports whether the
// cardinality threshold has been reached.
func (m *WAFIPCardinalityMiddleware) ObserveRequest(request *http.Request) (uint64, bool) {
	if m == nil || m.tracker == nil || request == nil {
		return 0, false
	}

	var addr netip.Addr
	if m.options.ResolveClientIP != nil {
		addr = m.options.ResolveClientIP(request)
	} else {
		addr = remoteClientAddr(request)
	}
	return m.tracker.ObserveAddr(addr)
}

// remoteClientAddr parses RemoteAddr without converting it to a string slice.
func remoteClientAddr(request *http.Request) netip.Addr {
	if request == nil {
		return netip.Addr{}
	}

	if endpoint, err := netip.ParseAddrPort(request.RemoteAddr); err == nil {
		return endpoint.Addr().Unmap().WithZone("")
	}
	if addr, err := netip.ParseAddr(request.RemoteAddr); err == nil {
		return addr.Unmap().WithZone("")
	}
	return netip.Addr{}
}

// Middleware wraps the next request handler with cardinality inspection.
func (m *WAFIPCardinalityMiddleware) Middleware(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		count, anomaly := m.ObserveRequest(request)
		if anomaly {
			if m.options.OnAnomaly != nil {
				m.options.OnAnomaly(request, count)
			}
			if m.options.RejectAnomalies {
				http.Error(writer, "client cardinality threshold exceeded", http.StatusTooManyRequests)
				return
			}
		}

		next.ServeHTTP(writer, request)
	})
}

// Wrap is an alias for Middleware.
func (m *WAFIPCardinalityMiddleware) Wrap(next http.Handler) http.Handler {
	return m.Middleware(next)
}

// Count returns the current distinct-client estimate.
func (m *WAFIPCardinalityMiddleware) Count() uint64 {
	if m == nil {
		return 0
	}
	return m.tracker.Count()
}

// Estimate returns the current unrounded distinct-client estimate.
func (m *WAFIPCardinalityMiddleware) Estimate() float64 {
	if m == nil {
		return 0
	}
	return m.tracker.Estimate()
}

// Reset starts a new cardinality observation window.
func (m *WAFIPCardinalityMiddleware) Reset() {
	if m != nil {
		m.tracker.Reset()
	}
}

// SetThreshold updates the anomaly threshold atomically.
func (m *WAFIPCardinalityMiddleware) SetThreshold(threshold uint64) {
	if m != nil {
		m.tracker.SetThreshold(threshold)
	}
}

// Tracker exposes the underlying sketch for distributed snapshot collection.
func (m *WAFIPCardinalityMiddleware) Tracker() *waf.IPCardinalityTracker {
	if m == nil {
		return nil
	}
	return m.tracker
}
