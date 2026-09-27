package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Aditya-9-6/DevProxy/pkg/certs"
	"github.com/Aditya-9-6/DevProxy/pkg/storage"
	"github.com/Aditya-9-6/DevProxy/web"
)

// Server provides the embedded HTTP management console and REST/WebSocket API.
type Server struct {
	store   *storage.Store
	hub     *Hub
	ca      *certs.CertificateAuthority
	addr    string
	httpSrv *http.Server
}

// NewServer creates a new dashboard Server instance.
func NewServer(addr string, store *storage.Store, hub *Hub, ca *certs.CertificateAuthority) *Server {
	return &Server{
		store: store,
		hub:   hub,
		ca:    ca,
		addr:  addr,
	}
}

// Start launches the HTTP server for the dashboard.
func (s *Server) Start() error {
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
	mux.HandleFunc("/api/ca.crt", s.handleDownloadCACert)

	s.httpSrv = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

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
