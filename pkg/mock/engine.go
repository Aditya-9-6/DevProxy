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

// ThrottlingProfile defines network condition simulation parameters.
type ThrottlingProfile struct {
	Name        string `json:"name"`
	DownloadBps int64  `json:"download_bps"`
	UploadBps   int64  `json:"upload_bps"`
	LatencyMs   int    `json:"latency_ms"`
	JitterMs    int    `json:"jitter_ms"`
	Offline     bool   `json:"offline"`
}

var StandardProfiles = map[string]ThrottlingProfile{
	"slow-3g": {
		Name:        "slow-3g",
		DownloadBps: 50 * 1024,
		UploadBps:   50 * 1024,
		LatencyMs:   400,
		JitterMs:    50,
		Offline:     false,
	},
	"fast-3g": {
		Name:        "fast-3g",
		DownloadBps: 200 * 1024,
		UploadBps:   90 * 1024,
		LatencyMs:   150,
		JitterMs:    20,
		Offline:     false,
	},
	"lte": {
		Name:        "lte",
		DownloadBps: 1250 * 1024,
		UploadBps:   625 * 1024,
		LatencyMs:   40,
		JitterMs:    10,
		Offline:     false,
	},
	"offline": {
		Name:    "offline",
		Offline: true,
	},
}

// Engine coordinates Map Local, Map Remote, Chaos injection, and Network Throttling.
type Engine struct {
	mu                sync.RWMutex
	mapLocal          []*MapLocalRule
	mapRemote         []*MapRemoteRule
	chaosRules        []*ChaosRule
	throttlingProfile string
}

// NewEngine initializes an empty mock & chaos engine.
func NewEngine() *Engine {
	return &Engine{
		mapLocal:          make([]*MapLocalRule, 0),
		mapRemote:         make([]*MapRemoteRule, 0),
		chaosRules:        make([]*ChaosRule, 0),
		throttlingProfile: "none",
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

// SetThrottlingProfile configures the active global network condition preset.
func (e *Engine) SetThrottlingProfile(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	clean := strings.ToLower(strings.TrimSpace(name))
	if clean == "" || clean == "none" {
		e.throttlingProfile = "none"
		return nil
	}

	if _, ok := StandardProfiles[clean]; !ok {
		return fmt.Errorf("unknown throttling profile: %s (supported: slow-3g, fast-3g, lte, offline, none)", name)
	}

	e.throttlingProfile = clean
	return nil
}

// GetThrottlingProfile returns the name of the currently active profile.
func (e *Engine) GetThrottlingProfile() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.throttlingProfile == "" {
		return "none"
	}
	return e.throttlingProfile
}

// ApplyChaos checks if network throttling, latency, or error simulation applies to the given URL.
func (e *Engine) ApplyChaos(rawURL string) (injectedError bool, status int, body string) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// 1. Check Global Network Throttling / Offline simulation
	if e.throttlingProfile != "" && e.throttlingProfile != "none" {
		if prof, ok := StandardProfiles[e.throttlingProfile]; ok {
			if prof.Offline {
				return true, http.StatusBadGateway, fmt.Sprintf(`{"error": "Simulated Network Offline (%s)", "target": "%s"}`, prof.Name, rawURL)
			}
			if prof.LatencyMs > 0 {
				delay := prof.LatencyMs
				if prof.JitterMs > 0 {
					j, _ := rand.Int(rand.Reader, big.NewInt(int64(prof.JitterMs*2)))
					delay += int(j.Int64()) - prof.JitterMs
					if delay < 0 {
						delay = 0
					}
				}
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}
		}
	}

	// 2. Check Targeted Chaos Rules
	for _, rule := range e.chaosRules {
		if !rule.Enabled || rule.regex == nil {
			continue
		}
		if rule.regex.MatchString(rawURL) {
			// Check Latency Delay
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

			// Check Simulated Error Rate
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

	prof := e.throttlingProfile
	if prof == "" {
		prof = "none"
	}

	return map[string]interface{}{
		"map_local":          e.mapLocal,
		"map_remote":         e.mapRemote,
		"chaos_rules":        e.chaosRules,
		"throttling_profile": prof,
	}
}
