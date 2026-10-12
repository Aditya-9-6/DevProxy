package handler

import (
	"net/http"

	"github.com/Aditya-9-6/DevProxy/pkg/limiter"
)

// RateLimitHandler wraps an HTTP handler and applies GCRA rate limiting.
type RateLimitHandler struct {
	next    http.Handler
	limiter *limiter.GCRALimiter
}

// NewRateLimitHandler creates a new rate limit middleware handler.
func NewRateLimitHandler(next http.Handler, rate float64, burst int) *RateLimitHandler {
	return &RateLimitHandler{
		next:    next,
		limiter: limiter.NewGCRALimiter(rate, burst),
	}
}

func (h *RateLimitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientKey := r.RemoteAddr
	if !h.limiter.Allow(clientKey) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	h.next.ServeHTTP(w, r)
}
