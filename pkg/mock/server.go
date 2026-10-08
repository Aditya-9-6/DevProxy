package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// StandaloneServer is a dedicated mock HTTP server that executes mock Engine rules.
type StandaloneServer struct {
	addr     string
	engine   *Engine
	server   *http.Server
	listener net.Listener
	mu       sync.RWMutex
}

// NewStandaloneServer creates a new mock server listening on the specified address.
func NewStandaloneServer(addr string, engine *Engine) *StandaloneServer {
	if engine == nil {
		engine = NewEngine()
	}

	s := &StandaloneServer{
		addr:   addr,
		engine: engine,
	}

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return s
}

// Start initiates listening and serving on the configured address.
func (s *StandaloneServer) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to bind mock server to %s: %w", s.addr, err)
	}

	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()

	return s.server.Serve(ln)
}

// Close gracefully stops the mock server.
func (s *StandaloneServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// Addr returns the bound network address.
func (s *StandaloneServer) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.addr
}

// ServeHTTP evaluates incoming requests against the mock engine.
func (s *StandaloneServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.String()
	if r.URL.Scheme == "" {
		host := r.Host
		if host == "" {
			host = "localhost"
		}
		rawURL = fmt.Sprintf("http://%s%s", host, r.URL.RequestURI())
	}

	// 1. Chaos injection
	if injected, status, body := s.engine.ApplyChaos(rawURL); injected {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-DevProxy-Chaos", "true")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
		return
	}

	// 2. Map Local mocking (test full URL first, then Path)
	rule, mockBody, err := s.engine.MatchMapLocal(rawURL)
	if (rule == nil || err != nil) && r.URL.Path != "" {
		rule, mockBody, err = s.engine.MatchMapLocal(r.URL.Path)
	}

	if rule != nil && err == nil {
		for k, v := range rule.Headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", rule.ContentType)
		w.Header().Set("X-DevProxy-Mock", "MapLocal")
		w.WriteHeader(rule.StatusCode)
		_, _ = w.Write(mockBody)
		return
	}

	// 3. Default mock fallback response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "mock_server_active",
		"method":  r.Method,
		"path":    r.URL.Path,
		"time":    time.Now().UTC().Format(time.RFC3339),
		"message": "DevProxy Standalone Mock Server. Add Map Local rules to customize endpoints.",
	})
}
