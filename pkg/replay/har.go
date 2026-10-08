package replay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/Aditya-9-6/DevProxy/pkg/storage"
)

// HARPlaybackServer serves mock responses deterministically based on recorded HAR 1.2 entries.
type HARPlaybackServer struct {
	addr            string
	server          *http.Server
	listener        net.Listener
	entries         []storage.HAREntry
	entriesByPath   map[string][]storage.HAREntry
	simulateLatency bool
	mu              sync.RWMutex
}

// NewHARPlaybackServer initializes a deterministic mock playback server from raw HAR JSON.
func NewHARPlaybackServer(addr string, harBytes []byte, simulateLatency bool) (*HARPlaybackServer, error) {
	var har storage.HAR
	if err := json.Unmarshal(harBytes, &har); err != nil {
		return nil, fmt.Errorf("failed to parse HAR JSON: %w", err)
	}

	entriesByPath := make(map[string][]storage.HAREntry)
	for _, entry := range har.Log.Entries {
		parsedURL, err := url.Parse(entry.Request.URL)
		path := ""
		if err == nil {
			path = parsedURL.Path
		}
		if path == "" {
			path = "/"
		}
		entriesByPath[path] = append(entriesByPath[path], entry)
	}

	ps := &HARPlaybackServer{
		addr:            addr,
		entries:         har.Log.Entries,
		entriesByPath:   entriesByPath,
		simulateLatency: simulateLatency,
	}

	ps.server = &http.Server{
		Addr:         addr,
		Handler:      ps,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return ps, nil
}

// NewHARPlaybackServerFromFile loads a HAR file from disk and initializes the server.
func NewHARPlaybackServerFromFile(addr string, filePath string, simulateLatency bool) (*HARPlaybackServer, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read HAR file %q: %w", filePath, err)
	}
	return NewHARPlaybackServer(addr, data, simulateLatency)
}

// Start begins listening on the configured address.
func (ps *HARPlaybackServer) Start() error {
	ln, err := net.Listen("tcp", ps.addr)
	if err != nil {
		return fmt.Errorf("failed to bind playback server on %s: %w", ps.addr, err)
	}
	ps.mu.Lock()
	ps.listener = ln
	ps.mu.Unlock()

	return ps.server.Serve(ln)
}

// Close gracefully terminates the playback server.
func (ps *HARPlaybackServer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return ps.server.Shutdown(ctx)
}

// Addr returns the network address the server is listening on.
func (ps *HARPlaybackServer) Addr() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if ps.listener != nil {
		return ps.listener.Addr().String()
	}
	return ps.addr
}

// ServeHTTP matches incoming requests against recorded HAR entries and replays them.
func (ps *HARPlaybackServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ps.mu.RLock()
	candidates, ok := ps.entriesByPath[r.URL.Path]
	ps.mu.RUnlock()

	if !ok || len(candidates) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "no matching HAR entry found",
			"method":  r.Method,
			"path":    r.URL.Path,
			"entries": len(ps.entries),
		})
		return
	}

	// Match by Method
	var matched *storage.HAREntry
	for i := range candidates {
		if strings.EqualFold(candidates[i].Request.Method, r.Method) {
			matched = &candidates[i]
			break
		}
	}

	if matched == nil {
		matched = &candidates[0]
	}

	// Simulate recorded timing latency if enabled
	if ps.simulateLatency && matched.Time > 0 {
		delay := time.Duration(matched.Time) * time.Millisecond
		if delay > 3*time.Second {
			delay = 3 * time.Second // Guard against excessive sleeps
		}
		time.Sleep(delay)
	}

	// Copy response headers
	for _, h := range matched.Response.Headers {
		// Skip hop-by-hop headers
		if strings.EqualFold(h.Name, "Transfer-Encoding") || strings.EqualFold(h.Name, "Connection") {
			continue
		}
		w.Header().Add(h.Name, h.Value)
	}

	statusCode := matched.Response.Status
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	w.WriteHeader(statusCode)

	// Decode and write body
	content := matched.Response.Content
	if content.Encoding == "base64" && content.Text != "" {
		decoded, err := base64.StdEncoding.DecodeString(content.Text)
		if err == nil {
			_, _ = w.Write(decoded)
			return
		}
	}

	if content.Text != "" {
		_, _ = w.Write([]byte(content.Text))
	}
}

// ExportTrafficEventsToHAR converts captured TrafficEvent instances to HAR 1.2 JSON.
func ExportTrafficEventsToHAR(events []*ringbuffer.TrafficEvent) ([]byte, error) {
	records := make([]*storage.RequestRecord, 0, len(events))
	for _, ev := range events {
		rec := &storage.RequestRecord{
			ID:          ev.ID,
			Timestamp:   ev.Timestamp,
			DurationMs:  float64(ev.Duration.Microseconds()) / 1000.0,
			Host:        ev.Host,
			Method:      ev.Method,
			URL:         ev.URL,
			Proto:       ev.Proto,
			ReqHeaders:  ev.ReqHeaders,
			ReqBody:     string(ev.ReqBody),
			StatusCode:  ev.StatusCode,
			RespHeaders: ev.RespHeaders,
			RespBody:    string(ev.RespBody),
		}
		records = append(records, rec)
	}
	return storage.GenerateHAR(records)
}
