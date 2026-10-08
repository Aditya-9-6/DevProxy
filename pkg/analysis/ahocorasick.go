package analysis

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// ACMatch represents a pattern occurrence identified in the payload.
type ACMatch struct {
	Signature SecretSignature
	EndIndex  int
	Snippet   string
}

// SecretSignature defines a signature to detect via Aho-Corasick.
type SecretSignature struct {
	ID          string
	Pattern     string
	Severity    string
	Category    string
	Description string
	Remediation string
}

// DefaultSecretSignatures returns standard API key and secret patterns.
func DefaultSecretSignatures() []SecretSignature {
	return []SecretSignature{
		{ID: "AWS-ACCESS-KEY", Pattern: "AKIA", Severity: SeverityCritical, Category: "EXPOSED_KEY", Description: "AWS IAM Access Key", Remediation: "Revoke AWS IAM credentials immediately."},
		{ID: "GITHUB-PAT", Pattern: "ghp_", Severity: SeverityCritical, Category: "EXPOSED_KEY", Description: "GitHub Personal Access Token", Remediation: "Revoke and rotate GitHub PAT token."},
		{ID: "STRIPE-LIVE-KEY", Pattern: "sk_live_", Severity: SeverityCritical, Category: "EXPOSED_KEY", Description: "Stripe Live Secret Key", Remediation: "Roll Stripe API keys immediately in dashboard."},
		{ID: "OPENAI-API-KEY", Pattern: "sk-proj-", Severity: SeverityCritical, Category: "EXPOSED_KEY", Description: "OpenAI Project API Key", Remediation: "Revoke OpenAI API key in platform console."},
		{ID: "RSA-PRIVATE-KEY", Pattern: "-----BEGIN RSA PRIVATE KEY-----", Severity: SeverityCritical, Category: "EXPOSED_SECRET", Description: "RSA Private Key Block", Remediation: "Rotate certificate private key."},
		{ID: "SLACK-BOT-TOKEN", Pattern: "xoxb-", Severity: SeverityHigh, Category: "EXPOSED_KEY", Description: "Slack Bot Token", Remediation: "Regenerate Slack bot OAuth token."},
		{ID: "GOOGLE-API-KEY", Pattern: "AIzaSy", Severity: SeverityHigh, Category: "EXPOSED_KEY", Description: "Google Cloud API Key", Remediation: "Restrict and rotate Google Cloud API key."},
	}
}

// ACNode represents a state in the Aho-Corasick automaton.
type ACNode struct {
	children map[byte]*ACNode
	fail     *ACNode
	outputs  []SecretSignature
}

// AhoCorasickMatcher implements the multi-pattern string searching algorithm.
type AhoCorasickMatcher struct {
	root *ACNode
	mu   sync.RWMutex
}

// NewAhoCorasickMatcher constructs an automaton from secret signatures.
func NewAhoCorasickMatcher(signatures []SecretSignature) *AhoCorasickMatcher {
	root := &ACNode{children: make(map[byte]*ACNode)}
	for _, sig := range signatures {
		curr := root
		for i := 0; i < len(sig.Pattern); i++ {
			b := sig.Pattern[i]
			if curr.children[b] == nil {
				curr.children[b] = &ACNode{children: make(map[byte]*ACNode)}
			}
			curr = curr.children[b]
		}
		curr.outputs = append(curr.outputs, sig)
	}

	queue := make([]*ACNode, 0)
	for _, child := range root.children {
		child.fail = root
		queue = append(queue, child)
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for b, child := range curr.children {
			f := curr.fail
			for f != nil && f.children[b] == nil {
				f = f.fail
			}
			if f != nil {
				child.fail = f.children[b]
			} else {
				child.fail = root
			}
			if child.fail != nil && len(child.fail.outputs) > 0 {
				child.outputs = append(child.outputs, child.fail.outputs...)
			}
			queue = append(queue, child)
		}
	}

	return &AhoCorasickMatcher{root: root}
}

// ScanBytes performs multi-pattern matching across data with sub-microsecond latency.
func (m *AhoCorasickMatcher) ScanBytes(data []byte) []ACMatch {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var matches []ACMatch
	curr := m.root
	for i, b := range data {
		for curr != m.root && curr.children[b] == nil {
			curr = curr.fail
		}
		if curr.children[b] == nil {
			curr = m.root
			continue
		}
		curr = curr.children[b]
		if len(curr.outputs) > 0 {
			for _, sig := range curr.outputs {
				start := i - len(sig.Pattern) + 1
				if start < 0 {
					start = 0
				}
				end := i + 24
				if end > len(data) {
					end = len(data)
				}
				rawSnippet := string(data[start:end])
				matches = append(matches, ACMatch{
					Signature: sig,
					EndIndex:  i,
					Snippet:   maskSecret(rawSnippet, sig.Pattern),
				})
			}
		}
	}
	return matches
}

func maskSecret(snippet, prefix string) string {
	idx := strings.Index(snippet, prefix)
	if idx == -1 {
		return snippet
	}
	revealed := idx + len(prefix)
	if len(snippet) <= revealed+4 {
		return snippet
	}
	maskStart := revealed + 4
	return snippet[:maskStart] + "****************"
}

// AhoCorasickSecretsRule wraps the matcher into the analysis Rule interface.
type AhoCorasickSecretsRule struct {
	matcher *AhoCorasickMatcher
}

// NewAhoCorasickSecretsRule returns a new rule instance.
func NewAhoCorasickSecretsRule() *AhoCorasickSecretsRule {
	return &AhoCorasickSecretsRule{
		matcher: NewAhoCorasickMatcher(DefaultSecretSignatures()),
	}
}

func (r *AhoCorasickSecretsRule) Name() string {
	return "Aho-Corasick Streaming Secret Matcher"
}

func (r *AhoCorasickSecretsRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding
	seen := make(map[string]bool)

	if len(event.ReqBody) > 0 {
		matches := r.matcher.ScanBytes(event.ReqBody)
		for _, m := range matches {
			key := m.Signature.ID + ":ReqBody"
			if !seen[key] {
				seen[key] = true
				findings = append(findings, &Finding{
					ID:          fmt.Sprintf("ac-%s-%s", event.ID, m.Signature.ID),
					RequestID:   event.ID,
					Timestamp:   time.Now(),
					Severity:    m.Signature.Severity,
					Category:    m.Signature.Category,
					RuleName:    r.Name(),
					Title:       fmt.Sprintf("%s in Request Body", m.Signature.ID),
					Description: m.Signature.Description,
					Evidence:    fmt.Sprintf("Matched %s: %s", m.Signature.Pattern, m.Snippet),
					Location:    "HTTP Request Body",
					Remediation: m.Signature.Remediation,
					URL:         event.URL,
					Method:      event.Method,
				})
			}
		}
	}

	if len(event.RespBody) > 0 {
		matches := r.matcher.ScanBytes(event.RespBody)
		for _, m := range matches {
			key := m.Signature.ID + ":RespBody"
			if !seen[key] {
				seen[key] = true
				findings = append(findings, &Finding{
					ID:          fmt.Sprintf("ac-resp-%s-%s", event.ID, m.Signature.ID),
					RequestID:   event.ID,
					Timestamp:   time.Now(),
					Severity:    m.Signature.Severity,
					Category:    m.Signature.Category,
					RuleName:    r.Name(),
					Title:       fmt.Sprintf("%s Leaked in Response Body", m.Signature.ID),
					Description: m.Signature.Description,
					Evidence:    fmt.Sprintf("Matched %s: %s", m.Signature.Pattern, m.Snippet),
					Location:    "HTTP Response Body",
					Remediation: m.Signature.Remediation,
					URL:         event.URL,
					Method:      event.Method,
				})
			}
		}
	}

	return findings
}

// SecretScanner provides backward compatibility for SecretScanner callers.
type SecretScanner struct {
	matcher *AhoCorasickMatcher
}

// NewSecretScanner creates a new SecretScanner.
func NewSecretScanner(patterns []string) *SecretScanner {
	sigs := make([]SecretSignature, 0, len(patterns))
	for _, p := range patterns {
		sigs = append(sigs, SecretSignature{
			ID:          p,
			Pattern:     p,
			Severity:    SeverityCritical,
			Category:    "EXPOSED_SECRET",
			Description: "Secret signature match: " + p,
		})
	}
	return &SecretScanner{
		matcher: NewAhoCorasickMatcher(sigs),
	}
}

func (s *SecretScanner) Name() string { return "SecretScanner" }

func (s *SecretScanner) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	rule := &AhoCorasickSecretsRule{matcher: s.matcher}
	return rule.Evaluate(event)
}
