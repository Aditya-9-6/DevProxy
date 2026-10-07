package dashboard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/contract"
	"github.com/Aditya-9-6/DevProxy/pkg/mock"
	"github.com/Aditya-9-6/DevProxy/pkg/replay"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"github.com/Aditya-9-6/DevProxy/pkg/storage"
	"github.com/Aditya-9-6/DevProxy/web"
)

// Server provides the embedded HTTP management console and REST/WebSocket API.
type Server struct {
	store             *storage.Store
	hub               *Hub
	ca                *certs.CertificateAuthority
	addr              string
	httpSrv           *http.Server
	mockEngine        *mock.Engine
	contractValidator *contract.Validator
	replayer          *replay.Replayer
	ringBuf           *ringbuffer.RingBuffer
}

// NewServer creates a new dashboard Server instance.
func NewServer(addr string, store *storage.Store, hub *Hub, ca *certs.CertificateAuthority) *Server {
	s := &Server{
		store:             store,
		hub:               hub,
		ca:                ca,
		addr:              addr,
		mockEngine:        mock.NewEngine(),
		contractValidator: contract.NewValidator(),
		replayer:          replay.NewReplayer(15*time.Second, true),
	}

	mux := http.NewServeMux()

	// Embedded UI
	mux.HandleFunc("/", s.handleIndex)

	// WebSockets
	mux.HandleFunc("/ws", s.hub.ServeWS)

	// REST API endpoints
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/requests", s.handleRequests)
	mux.HandleFunc("/api/requests/", s.handleRequestByID)
	mux.HandleFunc("/api/findings", s.handleFindings)
	mux.HandleFunc("/api/export/har", s.handleExportHAR)
	mux.HandleFunc("/api/import/har", s.handleImportHAR)
	mux.HandleFunc("/api/ca.crt", s.handleDownloadCACert)

	// Developer Superpower APIs
	mux.HandleFunc("/api/mocks", s.handleMocks)
	mux.HandleFunc("/api/mocks/local", s.handleAddMapLocal)
	mux.HandleFunc("/api/mocks/remote", s.handleAddMapRemote)
	mux.HandleFunc("/api/mocks/chaos", s.handleAddChaos)
	mux.HandleFunc("/api/mocks/throttling", s.handleMockThrottling)
	mux.HandleFunc("/api/replay", s.handleReplay)
	mux.HandleFunc("/api/contract/openapi", s.handleContractOpenAPI)
	mux.HandleFunc("/api/jwt/inspect", s.handleJWTInspect)

	s.httpSrv = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	return s
}

// SetMockEngine configures the mock & chaos engine.
func (s *Server) SetMockEngine(eng *mock.Engine) {
	s.mockEngine = eng
}

// SetRingBuffer configures the ring buffer for HAR import.
func (s *Server) SetRingBuffer(rb *ringbuffer.RingBuffer) {
	s.ringBuf = rb
}

// SetContractValidator configures the OpenAPI contract validator.
func (s *Server) SetContractValidator(cv *contract.Validator) {
	s.contractValidator = cv
}

// SetReplayer configures the request replay & diff engine.
func (s *Server) SetReplayer(r *replay.Replayer) {
	s.replayer = r
}

// Start launches the HTTP server for the dashboard.
func (s *Server) Start() error {
	return s.httpSrv.ListenAndServe()
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := web.Content.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Dashboard asset not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		if err := s.store.ClearAll(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"cleared"}`))
		return
	}

	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	query := r.URL.Query().Get("q")

	records, err := s.store.GetRecentRequests(limit, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if records == nil {
		w.Write([]byte("[]"))
		return
	}
	_ = json.NewEncoder(w).Encode(records)
}

func (s *Server) handleRequestByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/requests/")
	if id == "" {
		http.Error(w, "missing request id", http.StatusBadRequest)
		return
	}

	record, err := s.store.GetRequestByID(id)
	if err != nil {
		http.Error(w, "request not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(record)
}

func (s *Server) handleFindings(w http.ResponseWriter, r *http.Request) {
	limit := 100
	severity := r.URL.Query().Get("severity")

	findings, err := s.store.GetRecentFindings(limit, severity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if findings == nil {
		w.Write([]byte("[]"))
		return
	}
	_ = json.NewEncoder(w).Encode(findings)
}

func (s *Server) handleImportHAR(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	events, err := storage.ParseHAR(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse HAR: %v", err), http.StatusBadRequest)
		return
	}

	if s.ringBuf == nil {
		http.Error(w, "RingBuffer is not configured", http.StatusInternalServerError)
		return
	}

	count := 0
	for _, event := range events {
		if s.ringBuf.Push(event) {
			count++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"imported_events": count,
		"total_events":    len(events),
	})
}

func (s *Server) handleExportHAR(w http.ResponseWriter, r *http.Request) {
	records, err := s.store.GetRecentRequests(1000, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	harBytes, err := storage.GenerateHAR(records)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"devproxy-traffic.har\"")
	w.Write(harBytes)
}

func (s *Server) handleDownloadCACert(w http.ResponseWriter, r *http.Request) {
	if s.ca == nil {
		http.Error(w, "CA not initialized", http.StatusInternalServerError)
		return
	}

	pemBytes := s.ca.GetCACertificatePEM()
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", "attachment; filename=\"devproxy-ca.crt\"")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pemBytes)))
	w.Write(pemBytes)
}

// Close terminates the dashboard server.
func (s *Server) Close() error {
	if s.httpSrv != nil {
		return s.httpSrv.Close()
	}
	return nil
}

func (s *Server) handleMocks(w http.ResponseWriter, r *http.Request) {
	if s.mockEngine == nil {
		http.Error(w, "mock engine not initialized", http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.mockEngine.GetRulesSummary())
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id != "" {
			s.mockEngine.DeleteRule(id)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"deleted","id":"` + id + `"}`))
		} else {
			s.mockEngine.ClearAll()
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"all_cleared"}`))
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAddMapLocal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var rule mock.MapLocalRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "invalid json body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.mockEngine.AddMapLocal(&rule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(rule)
}

func (s *Server) handleAddMapRemote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var rule mock.MapRemoteRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "invalid json body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.mockEngine.AddMapRemote(&rule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(rule)
}

func (s *Server) handleAddChaos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var rule mock.ChaosRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "invalid json body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.mockEngine.AddChaosRule(&rule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(rule)
}

func (s *Server) handleContractOpenAPI(w http.ResponseWriter, r *http.Request) {
	if s.contractValidator == nil {
		http.Error(w, "contract validator not initialized", http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.contractValidator.GetInfo())
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			http.Error(w, "empty specification body", http.StatusBadRequest)
			return
		}
		if err := s.contractValidator.LoadSpec(body); err != nil {
			http.Error(w, "failed to parse spec: "+err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.contractValidator.GetInfo())
	case http.MethodDelete:
		s.contractValidator.ClearSpec()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"cleared"}`))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleJWTInspect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		http.Error(w, "missing or invalid 'token' in request body", http.StatusBadRequest)
		return
	}
	details, _ := analysis.ParseAndInspectJWT(req.Token, "inspect", "manual", "POST", "JWT Inspector")
	if details == nil {
		http.Error(w, "failed to parse JWT token (invalid structure)", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(details)
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		RequestID string              `json:"request_id"`
		Method    string              `json:"method,omitempty"`
		URL       string              `json:"url,omitempty"`
		Headers   map[string][]string `json:"headers,omitempty"`
		Body      string              `json:"body,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if payload.RequestID == "" {
		http.Error(w, "missing request_id in request body", http.StatusBadRequest)
		return
	}

	orig, err := s.store.GetRequestByID(payload.RequestID)
	if err != nil || orig == nil {
		http.Error(w, "original request not found: "+payload.RequestID, http.StatusNotFound)
		return
	}

	var override *replay.ReplayOverride
	if payload.Method != "" || payload.URL != "" || payload.Headers != nil || payload.Body != "" {
		override = &replay.ReplayOverride{
			Method:  payload.Method,
			URL:     payload.URL,
			Headers: payload.Headers,
			Body:    payload.Body,
		}
	}

	if s.replayer == nil {
		s.replayer = replay.NewReplayer(15*time.Second, true)
	}

	result, err := s.replayer.Replay(orig, override)
	if err != nil {
		http.Error(w, "replay execution failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *Server) handleMockThrottling(w http.ResponseWriter, r *http.Request) {
	if s.mockEngine == nil {
		http.Error(w, "mock engine not initialized", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"profile": s.mockEngine.GetThrottlingProfile(),
		})
	case http.MethodPost:
		var req struct {
			Profile string `json:"profile"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.mockEngine.SetThrottlingProfile(req.Profile); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"profile": s.mockEngine.GetThrottlingProfile(),
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
