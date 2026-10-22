package analysis

import (
	"fmt"
	"strings"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// SecretSignature defines a sensitive token pattern for Aho-Corasick multi-string scanning.
type SecretSignature struct {
	ID          string `json:"id"`
	Pattern     string `json:"pattern"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// AhoCorasickNode represents a node in the Aho-Corasick automaton trie.
type AhoCorasickNode struct {
	children map[byte]*AhoCorasickNode
	fail     *AhoCorasickNode
	outputs  []*SecretSignature
}

func newACNode() *AhoCorasickNode {
	return &AhoCorasickNode{
		children: make(map[byte]*AhoCorasickNode),
		outputs:  make([]*SecretSignature, 0),
	}
}

// AhoCorasickMatcher provides deterministic O(N + M) streaming multi-keyword secret matching.
type AhoCorasickMatcher struct {
	root       *AhoCorasickNode
	signatures []*SecretSignature
}

// NewAhoCorasickMatcher builds and compiles an automaton from a list of signatures.
func NewAhoCorasickMatcher(signatures []*SecretSignature) *AhoCorasickMatcher {
	m := &AhoCorasickMatcher{
		root:       newACNode(),
		signatures: signatures,
	}
	m.buildTrie()
	m.buildFailureLinks()
	return m
}

// DefaultSecretSignatures provides industry-standard secret patterns for zero-latency detection.
func DefaultSecretSignatures() []*SecretSignature {
	return []*SecretSignature{
		{
			ID:          "AWS-ACCESS-KEY",
			Pattern:     "AKIA",
			Category:    "EXPOSED_SECRET",
			Severity:    SeverityCritical,
			Description: "Detected potential AWS IAM Access Key ID prefix (AKIA)",
			Remediation: "Revoke AWS key immediately in IAM console and switch to AWS Secrets Manager or IAM Roles.",
		},
		{
			ID:          "GITHUB-PAT",
			Pattern:     "ghp_",
			Category:    "EXPOSED_SECRET",
			Severity:    SeverityCritical,
			Description: "Detected GitHub Personal Access Token prefix (ghp_)",
			Remediation: "Revoke the GitHub PAT from GitHub Settings > Developer Settings > Personal Access Tokens.",
		},
		{
			ID:          "GITHUB-OAUTH",
			Pattern:     "gho_",
			Category:    "EXPOSED_SECRET",
			Severity:    SeverityCritical,
			Description: "Detected GitHub OAuth Access Token prefix (gho_)",
			Remediation: "Revoke OAuth token and re-authenticate securely via backend OAuth flow.",
		},
		{
			ID:          "GITHUB-PAT-FINEGRAINED",
			Pattern:     "github_pat_",
			Category:    "EXPOSED_SECRET",
			Severity:    SeverityCritical,
			Description: "Detected GitHub Fine-Grained Personal Access Token prefix (github_pat_)",
			Remediation: "Revoke this token from GitHub settings.",
		},
		{
			ID:          "SLACK-BOT-TOKEN",
			Pattern:     "xoxb-",
			Category:    "EXPOSED_SECRET",
			Severity:    SeverityCritical,
			Description: "Detected Slack Bot OAuth Token prefix (xoxb-)",
			Remediation: "Rotate Slack application bot credentials in api.slack.com apps portal.",
		},
		{
			ID:          "SLACK-USER-TOKEN",
			Pattern:     "xoxp-",
			Category:    "EXPOSED_SECRET",
			Severity:    SeverityHigh,
			Description: "Detected Slack User OAuth Token prefix (xoxp-)",
			Remediation: "Rotate compromised Slack user access token.",
		},
		{
			ID:          "OPENAI-API-KEY",
			Pattern:     "sk-proj-",
			Category:    "EXPOSED_SECRET",
			Severity:    SeverityCritical,
			Description: "Detected OpenAI Project API Key (sk-proj-)",
			Remediation: "Rotate key immediately at platform.openai.com/api-keys.",
		},
		{
			ID:          "STRIPE-LIVE-KEY",
			Pattern:     "sk_live_",
			Category:    "EXPOSED_SECRET",
			Severity:    SeverityCritical,
			Description: "Detected Stripe Live Secret API Key (sk_live_)",
			Remediation: "Roll the compromised Stripe secret key in the Stripe Dashboard immediately.",
		},
		{
			ID:          "STRIPE-RESTRICTED-KEY",
			Pattern:     "rk_live_",
			Category:    "EXPOSED_SECRET",
			Severity:    SeverityCritical,
			Description: "Detected Stripe Restricted Live API Key (rk_live_)",
			Remediation: "Revoke the restricted API key in Stripe Dashboard.",
		},
		{
			ID:          "GOOGLE-API-KEY",
			Pattern:     "AIzaSy",
			Category:    "EXPOSED_SECRET",
			Severity:    SeverityHigh,
			Description: "Detected Google Cloud / Firebase API Key prefix (AIzaSy)",
			Remediation: "Restrict this key to specific HTTP referrers / IP ranges in Google Cloud Console.",
		},
		{
			ID:          "PRIVATE-KEY-RSA",
			Pattern:     "-----BEGIN RSA PRIVATE KEY-----",
			Category:    "EXPOSED_KEY",
			Severity:    SeverityCritical,
			Description: "Detected unencrypted RSA Private Key header",
			Remediation: "Never transmit private keys in plaintext over HTTP/HTTPS. Rotate keypair immediately.",
		},
		{
			ID:          "PRIVATE-KEY-PKCS8",
			Pattern:     "-----BEGIN PRIVATE KEY-----",
			Category:    "EXPOSED_KEY",
			Severity:    SeverityCritical,
			Description: "Detected unencrypted PKCS#8 Private Key header",
			Remediation: "Store private keys in a Hardware Security Module (HSM) or secure vault.",
		},
	}
}

func (m *AhoCorasickMatcher) buildTrie() {
	for _, sig := range m.signatures {
		curr := m.root
		patternBytes := []byte(sig.Pattern)
		for _, b := range patternBytes {
			next, ok := curr.children[b]
			if !ok {
				next = newACNode()
				curr.children[b] = next
			}
			curr = next
		}
		curr.outputs = append(curr.outputs, sig)
	}
}

func (m *AhoCorasickMatcher) buildFailureLinks() {
	queue := make([]*AhoCorasickNode, 0)

	// Step 1: Set failure links for depth 1 nodes to root
	for _, child := range m.root.children {
		child.fail = m.root
		queue = append(queue, child)
	}

	// Step 2: BFS for deeper nodes
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for b, child := range curr.children {
			f := curr.fail
			for f != nil && f.children[b] == nil {
				f = f.fail
			}
			if f == nil {
				child.fail = m.root
			} else {
				child.fail = f.children[b]
			}

			// Propagate dictionary match outputs
			if child.fail != nil && len(child.fail.outputs) > 0 {
				child.outputs = append(child.outputs, child.fail.outputs...)
			}

			queue = append(queue, child)
		}
	}
}

// ACMatch records the detection of a secret pattern in the scanned payload.
type ACMatch struct {
	Signature *SecretSignature
	EndIndex  int
	Snippet   string
}

// ScanBytes performs deterministic linear O(N) multi-pattern search over raw bytes without allocation.
func (m *AhoCorasickMatcher) ScanBytes(data []byte) []ACMatch {
	if len(data) == 0 {
		return nil
	}

	var matches []ACMatch
	curr := m.root

	for i, b := range data {
		for curr != nil && curr.children[b] == nil {
			curr = curr.fail
		}
		if curr == nil {
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
				// Extract safe evidence snippet (masked)
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
	// Mask everything after prefix + 4 chars
	maskStart := revealed + 4
	return snippet[:maskStart] + "****************"
}

// AhoCorasickSecretsRule wraps the matcher into the standard DevProxy analysis.Rule interface.
type AhoCorasickSecretsRule struct {
	matcher *AhoCorasickMatcher
}

// NewAhoCorasickSecretsRule creates a new rule instance.
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

	// Scan Request Body
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

	// Scan Response Body
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

	// Scan URL query strings
	if len(event.URL) > 0 {
		matches := r.matcher.ScanBytes([]byte(event.URL))
		for _, m := range matches {
			key := m.Signature.ID + ":URL"
			if !seen[key] {
				seen[key] = true
				findings = append(findings, &Finding{
					ID:          fmt.Sprintf("ac-url-%s-%s", event.ID, m.Signature.ID),
					RequestID:   event.ID,
					Timestamp:   time.Now(),
					Severity:    m.Signature.Severity,
					Category:    m.Signature.Category,
					RuleName:    r.Name(),
					Title:       fmt.Sprintf("%s in URL Query String", m.Signature.ID),
					Description: m.Signature.Description,
					Evidence:    fmt.Sprintf("Matched %s: %s", m.Signature.Pattern, m.Snippet),
					Location:    "HTTP Request URL",
					Remediation: m.Signature.Remediation,
					URL:         event.URL,
					Method:      event.Method,
				})
			}
		}
	}

	return findings
}
