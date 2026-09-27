package mock

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MapLocalRule serves a local file or inline payload instead of hitting the upstream server.
type MapLocalRule struct {
	ID          string            `json:"id"`
	Enabled     bool              `json:"enabled"`
	Pattern     string            `json:"pattern"` // regex matching URL or path
	regex       *regexp.Regexp    `json:"-"`
	FilePath    string            `json:"file_path,omitempty"` // local file to serve
	InlineBody  string            `json:"inline_body,omitempty"`
	StatusCode  int               `json:"status_code"`
	ContentType string            `json:"content_type"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// MapRemoteRule redirects requests matching a pattern to a different remote or local destination.
type MapRemoteRule struct {
	ID         string         `json:"id"`
	Enabled    bool           `json:"enabled"`
	Pattern    string         `json:"pattern"` // regex matching source URL
	regex      *regexp.Regexp `json:"-"`
	TargetHost string         `json:"target_host"` // target host/base (e.g. localhost:3000)
	TargetURL  string         `json:"target_url,omitempty"`
}

// ChaosRule injects latency, jitter, or simulated errors to test client resilience.
type ChaosRule struct {
	ID          string         `json:"id"`
	Enabled     bool           `json:"enabled"`
	Pattern     string         `json:"pattern"` // regex matching target URLs (or ".*" for all)
	regex       *regexp.Regexp `json:"-"`
	DelayMs     int            `json:"delay_ms"`     // delay in milliseconds
	JitterMs    int            `json:"jitter_ms"`    // random variance +/- jitter
	ErrorRate   float64        `json:"error_rate"`   // probability (0.0 to 1.0) of simulated failure
	ErrorStatus int            `json:"error_status"` // e.g. 500, 502, 503
	ErrorBody   string         `json:"error_body"`
}

// Engine coordinates Map Local, Map Remote, and Chaos injection.
type Engine struct {
	mu         sync.RWMutex
	mapLocal   []*MapLocalRule
	mapRemote  []*MapRemoteRule
	chaosRules []*ChaosRule
}

// NewEngine initializes an empty mock & chaos engine.
func NewEngine() *Engine {
	return &Engine{
		mapLocal:   make([]*MapLocalRule, 0),
		mapRemote:  make([]*MapRemoteRule, 0),
		chaosRules: make([]*ChaosRule, 0),
	}
}

// AddMapLocal registers a new Map Local rule.
func (e *Engine) AddMapLocal(rule *MapLocalRule) error {
	if rule.ID == "" {
		rule.ID = "local-" + uuid.NewString()[:8]
	}
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern regex: %w", err)
	}
	rule.regex = re
	if rule.StatusCode == 0 {
		rule.StatusCode = http.StatusOK
	}
	if rule.ContentType == "" {
		rule.ContentType = "application/json; charset=utf-8"
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.mapLocal = append(e.mapLocal, rule)
	return nil
}

// AddMapRemote registers a new Map Remote rule.
func (e *Engine) AddMapRemote(rule *MapRemoteRule) error {
	if rule.ID == "" {
		rule.ID = "remote-" + uuid.NewString()[:8]
	}
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern regex: %w", err)
	}
	rule.regex = re

	e.mu.Lock()
	defer e.mu.Unlock()
	e.mapRemote = append(e.mapRemote, rule)
	return nil
}

// AddChaosRule registers a new Chaos / Latency injection rule.
func (e *Engine) AddChaosRule(rule *ChaosRule) error {
	if rule.ID == "" {
		rule.ID = "chaos-" + uuid.NewString()[:8]
	}
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern regex: %w", err)
	}
	rule.regex = re
	if rule.ErrorStatus == 0 {
		rule.ErrorStatus = http.StatusServiceUnavailable
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.chaosRules = append(e.chaosRules, rule)
	return nil
}

// DeleteRule removes a rule with the specified ID across all rule types.
func (e *Engine) DeleteRule(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, r := range e.mapLocal {
		if r.ID == id {
			e.mapLocal = append(e.mapLocal[:i], e.mapLocal[i+1:]...)
			return true
		}
	}
	for i, r := range e.mapRemote {
		if r.ID == id {
			e.mapRemote = append(e.mapRemote[:i], e.mapRemote[i+1:]...)
			return true
		}
	}
	for i, r := range e.chaosRules {
		if r.ID == id {
			e.chaosRules = append(e.chaosRules[:i], e.chaosRules[i+1:]...)
			return true
		}
	}
	return false
}

// ClearAll removes all registered rules.
func (e *Engine) ClearAll() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.mapLocal = make([]*MapLocalRule, 0)
	e.mapRemote = make([]*MapRemoteRule, 0)
	e.chaosRules = make([]*ChaosRule, 0)
}

// ApplyChaos checks if latency or error simulation applies to the given URL.
func (e *Engine) ApplyChaos(rawURL string) (injectedError bool, status int, body string) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.chaosRules {
		if !rule.Enabled || rule.regex == nil {
			continue
		}
		if rule.regex.MatchString(rawURL) {
			// 1. Check Latency Delay
			if rule.DelayMs > 0 {
				delay := rule.DelayMs
				if rule.JitterMs > 0 {
					j, _ := rand.Int(rand.Reader, big.NewInt(int64(rule.JitterMs*2)))
					delay += int(j.Int64()) - rule.JitterMs
					if delay < 0 {
						delay = 0
					}
				}
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}

			// 2. Check Simulated Error Rate
			if rule.ErrorRate > 0 {
				n, _ := rand.Int(rand.Reader, big.NewInt(1000))
				prob := float64(n.Int64()) / 1000.0
				if prob <= rule.ErrorRate {
					errBody := rule.ErrorBody
					if errBody == "" {
						errBody = fmt.Sprintf(`{"error": "Simulated Chaos Error (%d)", "target": "%s"}`, rule.ErrorStatus, rawURL)
					}
					return true, rule.ErrorStatus, errBody
				}
			}
		}
	}
	return false, 0, ""
}

// MatchMapLocal checks if a URL matches any active Map Local rules.
func (e *Engine) MatchMapLocal(rawURL string) (*MapLocalRule, []byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.mapLocal {
		if !rule.Enabled || rule.regex == nil {
			continue
		}
		if rule.regex.MatchString(rawURL) {
			var body []byte
			if rule.FilePath != "" {
				data, err := os.ReadFile(rule.FilePath)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to read mock file %s: %w", rule.FilePath, err)
				}
				body = data
			} else {
				body = []byte(rule.InlineBody)
			}
			return rule, body, nil
		}
	}
	return nil, nil, nil
}

// MatchMapRemote rewrites host/URL if matching any active Map Remote rules.
func (e *Engine) MatchMapRemote(rawURL, host string) (newURL string, newHost string, matched bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, rule := range e.mapRemote {
		if !rule.Enabled || rule.regex == nil {
			continue
		}
		if rule.regex.MatchString(rawURL) {
			target := rule.TargetHost
			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
				// Strip scheme for host
				parts := strings.SplitN(target, "://", 2)
				target = parts[1]
			}
			return strings.Replace(rawURL, host, target, 1), target, true
		}
	}
	return rawURL, host, false
}

// GetRulesSummary returns current configuration for the dashboard.
func (e *Engine) GetRulesSummary() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return map[string]interface{}{
		"map_local":   e.mapLocal,
		"map_remote":  e.mapRemote,
		"chaos_rules": e.chaosRules,
	}
}
