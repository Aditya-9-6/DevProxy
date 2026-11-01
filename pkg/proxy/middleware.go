package proxy

import (
	"net/http"
)

// Middleware defines the interface for HTTP request/response interception.
type Middleware func(http.Handler) http.Handler

// Chain combines multiple middlewares into a single Middleware.
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}
