package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/analysis"
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	_ "modernc.org/sqlite"
)

// Store provides an ephemeral in-memory SQLite database for captured requests and security findings.
type Store struct {
	db         *sql.DB
	mu         sync.RWMutex
	maxRecords int
	reqCount   sync.Mutex
	counter    uint64
}

// StatsSummary contains high-level metrics for the dashboard.
type StatsSummary struct {
	TotalRequests   int64            `json:"total_requests"`
	TotalFindings   int64            `json:"total_findings"`
	SeverityCounts  map[string]int64 `json:"severity_counts"`
	AvgLatencyMs    float64          `json:"avg_latency_ms"`
	TotalBytesTrans int64            `json:"total_bytes"`
}

// RequestRecord represents stored HTTP transaction metadata.
type RequestRecord struct {
	ID           string              `json:"id"`
	Timestamp    time.Time           `json:"timestamp"`
	DurationMs   float64             `json:"duration_ms"`
	ClientIP     string              `json:"client_ip"`
	Scheme       string              `json:"scheme"`
	Host         string              `json:"host"`
	Method       string              `json:"method"`
	Path         string              `json:"path"`
	URL          string              `json:"url"`
	Proto        string              `json:"proto"`
	StatusCode   int                 `json:"status_code"`
	ReqHeaders   map[string][]string `json:"req_headers"`
	ReqBody      string              `json:"req_body"`
	RespHeaders  map[string][]string `json:"resp_headers"`
	RespBody     string              `json:"resp_body"`
	TLS          bool                `json:"tls"`
	FindingCount int                 `json:"finding_count"`
	Findings     []*analysis.Finding `json:"findings,omitempty"`
}

// NewStore initializes a new in-memory SQLite store with schema and indexes.
func NewStore() (*Store, error) {
	// Use shared cache in-memory SQLite
	dsn := "file:devproxy_mem?mode=memory&cache=shared"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite in-memory db: %w", err)
	}

	// Max connections: SQLite in-memory works best with shared cache and configured pool
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return s, nil
}

func (s *Store) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS requests (
			id TEXT PRIMARY KEY,
			timestamp DATETIME,
			duration_ns INTEGER,
			client_ip TEXT,
			scheme TEXT,
			host TEXT,
			method TEXT,
			path TEXT,
			url TEXT,
			proto TEXT,
			status_code INTEGER,
			req_headers TEXT,
			req_body TEXT,
			resp_headers TEXT,
			resp_body TEXT,
			tls INTEGER,
			finding_count INTEGER DEFAULT 0
		);`,
		`CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp DESC);`,
		`CREATE TABLE IF NOT EXISTS findings (
			id TEXT PRIMARY KEY,
			request_id TEXT,
			timestamp DATETIME,
			severity TEXT,
			category TEXT,
			rule_name TEXT,
			title TEXT,
			description TEXT,
			evidence TEXT,
			location TEXT,
			remediation TEXT,
			url TEXT,
			method TEXT,
			FOREIGN KEY (request_id) REFERENCES requests(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_findings_timestamp ON findings(timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_findings_req_id ON findings(request_id);`,
		`CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings(severity);`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// SaveTransaction inserts a request event along with any findings into SQLite.
func (s *Store) SaveTransaction(event *ringbuffer.TrafficEvent, findings []*analysis.Finding) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reqHeadersJSON, _ := json.Marshal(event.ReqHeaders)
	respHeadersJSON, _ := json.Marshal(event.RespHeaders)

	tlsInt := 0
	if event.TLS {
		tlsInt = 1
	}

	// Limit stored preview bodies in DB to 64KB each to keep memory light
	reqBodyStr := string(event.ReqBody)
	if len(reqBodyStr) > 65536 {
		reqBodyStr = reqBodyStr[:65536] + "\n...[truncated]"
	}

	respBodyStr := string(event.RespBody)
	if len(respBodyStr) > 65536 {
		respBodyStr = respBodyStr[:65536] + "\n...[truncated]"
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertReq := `INSERT OR REPLACE INTO requests (
		id, timestamp, duration_ns, client_ip, scheme, host, method, path, url, proto,
		status_code, req_headers, req_body, resp_headers, resp_body, tls, finding_count
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err = tx.Exec(insertReq,
		event.ID,
		event.Timestamp,
		event.Duration.Nanoseconds(),
		event.ClientIP,
		event.Scheme,
		event.Host,
		event.Method,
		event.Path,
		event.URL,
		event.Proto,
		event.StatusCode,
		string(reqHeadersJSON),
		reqBodyStr,
		string(respHeadersJSON),
		respBodyStr,
		tlsInt,
		len(findings),
	)
	if err != nil {
		return err
	}

	insertFinding := `INSERT OR REPLACE INTO findings (
		id, request_id, timestamp, severity, category, rule_name, title, description, evidence, location, remediation, url, method
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	for idx, f := range findings {
		if f.ID == "" {
			f.ID = fmt.Sprintf("%s-f%d", event.ID, idx+1)
		}
		_, err = tx.Exec(insertFinding,
			f.ID,
			event.ID,
			f.Timestamp,
			f.Severity,
			f.Category,
			f.RuleName,
			f.Title,
			f.Description,
			f.Evidence,
			f.Location,
			f.Remediation,
			f.URL,
			f.Method,
		)
		if err != nil {
			return err
		}
	}

	s.reqCount.Lock()
	s.counter++
	cnt := s.counter
	s.reqCount.Unlock()

	if cnt%50 == 0 {
		go func() {
			_ = s.PruneOldRecords(s.maxRecords)
		}()
	}

	return tx.Commit()
}

// PruneOldRecords deletes records exceeding keepCount to prevent memory growth in long sessions.
func (s *Store) PruneOldRecords(keepCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if keepCount <= 0 {
		keepCount = 5000
	}

	pruneFindings := `DELETE FROM findings WHERE request_id IN (
		SELECT id FROM requests WHERE id NOT IN (
			SELECT id FROM requests ORDER BY timestamp DESC LIMIT ?
		)
	);`
	_, _ = s.db.Exec(pruneFindings, keepCount)

	pruneReqs := `DELETE FROM requests WHERE id NOT IN (
		SELECT id FROM requests ORDER BY timestamp DESC LIMIT ?
	);`
	_, err := s.db.Exec(pruneReqs, keepCount)
	return err
}

// ClearAll deletes all stored requests and findings.
func (s *Store) ClearAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`DELETE FROM findings;`); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM requests;`)
	return err
}

// GetRecentRequests retrieves the latest requests up to limit.
func (s *Store) GetRecentRequests(limit int, query string) ([]*RequestRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 50
	}

	baseQuery := `SELECT id, timestamp, duration_ns, client_ip, scheme, host, method, path, url, proto, status_code, finding_count, tls
		FROM requests`
	var rows *sql.Rows
	var err error

	if query != "" {
		likePattern := "%" + query + "%"
		q := baseQuery + ` WHERE url LIKE ? OR method LIKE ? OR host LIKE ? ORDER BY timestamp DESC LIMIT ?;`
		rows, err = s.db.Query(q, likePattern, likePattern, likePattern, limit)
	} else {
		q := baseQuery + ` ORDER BY timestamp DESC LIMIT ?;`
		rows, err = s.db.Query(q, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*RequestRecord
	for rows.Next() {
		var r RequestRecord
		var durationNs int64
		var tlsInt int
		err := rows.Scan(
			&r.ID, &r.Timestamp, &durationNs, &r.ClientIP, &r.Scheme, &r.Host,
			&r.Method, &r.Path, &r.URL, &r.Proto, &r.StatusCode, &r.FindingCount, &tlsInt,
		)
		if err != nil {
			return nil, err
		}
		r.DurationMs = float64(durationNs) / 1e6
		r.TLS = (tlsInt == 1)
		records = append(records, &r)
	}

	return records, nil
}

// GetRequestByID retrieves full request and response pair along with any findings.
func (s *Store) GetRequestByID(id string) (*RequestRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q := `SELECT id, timestamp, duration_ns, client_ip, scheme, host, method, path, url, proto, status_code,
		req_headers, req_body, resp_headers, resp_body, tls, finding_count
		FROM requests WHERE id = ?;`

	var r RequestRecord
	var durationNs int64
	var reqHeadersJSON, respHeadersJSON string
	var tlsInt int

	err := s.db.QueryRow(q, id).Scan(
		&r.ID, &r.Timestamp, &durationNs, &r.ClientIP, &r.Scheme, &r.Host,
		&r.Method, &r.Path, &r.URL, &r.Proto, &r.StatusCode,
		&reqHeadersJSON, &r.ReqBody, &respHeadersJSON, &r.RespBody, &tlsInt, &r.FindingCount,
	)
	if err != nil {
		return nil, err
	}

	r.DurationMs = float64(durationNs) / 1e6
	r.TLS = (tlsInt == 1)
	_ = json.Unmarshal([]byte(reqHeadersJSON), &r.ReqHeaders)
	_ = json.Unmarshal([]byte(respHeadersJSON), &r.RespHeaders)

	// Fetch findings
	fq := `SELECT id, request_id, timestamp, severity, category, rule_name, title, description, evidence, location, remediation, url, method
		FROM findings WHERE request_id = ? ORDER BY timestamp ASC;`
	fRows, err := s.db.Query(fq, id)
	if err == nil {
		defer fRows.Close()
		for fRows.Next() {
			var f analysis.Finding
			if err := fRows.Scan(&f.ID, &f.RequestID, &f.Timestamp, &f.Severity, &f.Category, &f.RuleName, &f.Title, &f.Description, &f.Evidence, &f.Location, &f.Remediation, &f.URL, &f.Method); err == nil {
				r.Findings = append(r.Findings, &f)
			}
		}
	}

	return &r, nil
}

// GetRecentFindings returns latest security findings.
func (s *Store) GetRecentFindings(limit int, severity string) ([]*analysis.Finding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	var rows *sql.Rows
	var err error

	if severity != "" && severity != "ALL" {
		q := `SELECT id, request_id, timestamp, severity, category, rule_name, title, description, evidence, location, remediation, url, method
			FROM findings WHERE severity = ? ORDER BY timestamp DESC LIMIT ?;`
		rows, err = s.db.Query(q, severity, limit)
	} else {
		q := `SELECT id, request_id, timestamp, severity, category, rule_name, title, description, evidence, location, remediation, url, method
			FROM findings ORDER BY timestamp DESC LIMIT ?;`
		rows, err = s.db.Query(q, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var findings []*analysis.Finding
	for rows.Next() {
		var f analysis.Finding
		err := rows.Scan(&f.ID, &f.RequestID, &f.Timestamp, &f.Severity, &f.Category, &f.RuleName, &f.Title, &f.Description, &f.Evidence, &f.Location, &f.Remediation, &f.URL, &f.Method)
		if err != nil {
			return nil, err
		}
		findings = append(findings, &f)
	}

	return findings, nil
}

// GetStats returns summary metrics for the dashboard.
func (s *Store) GetStats() (*StatsSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &StatsSummary{
		SeverityCounts: map[string]int64{
			analysis.SeverityCritical: 0,
			analysis.SeverityHigh:     0,
			analysis.SeverityMedium:   0,
			analysis.SeverityLow:      0,
			analysis.SeverityInfo:     0,
		},
	}

	_ = s.db.QueryRow(`SELECT COUNT(*), COALESCE(AVG(duration_ns) / 1000000.0, 0) FROM requests;`).
		Scan(&stats.TotalRequests, &stats.AvgLatencyMs)

	_ = s.db.QueryRow(`SELECT COUNT(*) FROM findings;`).Scan(&stats.TotalFindings)

	rows, err := s.db.Query(`SELECT severity, COUNT(*) FROM findings GROUP BY severity;`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sev string
			var count int64
			if err := rows.Scan(&sev, &count); err == nil {
				stats.SeverityCounts[sev] = count
			}
		}
	}

	return stats, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
