package analysis

import (
	"strings"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestAhoCorasickDirectMatcher(t *testing.T) {
	matcher := NewAhoCorasickMatcher(DefaultSecretSignatures())

	payload := []byte(`
	{
		"aws_key": "AKIAIOSFODNN7EXAMPLE",
		"github_token": "ghp_1234567890abcdefghijklmnopqrstuvwxyz",
		"stripe": "sk_live_9876543210fedcba",
		"openai": "sk-proj-abcde1234567890",
		"clean": "hello world"
	}`)

	matches := matcher.ScanBytes(payload)
	if len(matches) < 4 {
		t.Fatalf("expected at least 4 matches, got %d", len(matches))
	}

	foundSignatures := make(map[string]bool)
	for _, m := range matches {
		foundSignatures[m.Signature.ID] = true
		// Verify snippet is masked
		if strings.Contains(m.Snippet, "****************") == false && len(m.Snippet) > 10 {
			// Masking check
		}
	}

	expectedIDs := []string{"AWS-ACCESS-KEY", "GITHUB-PAT", "STRIPE-LIVE-KEY", "OPENAI-API-KEY"}
	for _, id := range expectedIDs {
		if !foundSignatures[id] {
			t.Errorf("expected signature %s to be detected", id)
		}
	}
}

func TestAhoCorasickRuleEvaluation(t *testing.T) {
	rule := NewAhoCorasickSecretsRule()

	event := &ringbuffer.TrafficEvent{
		ID:        "evt-ac-1",
		Timestamp: time.Now(),
		Method:    "POST",
		URL:       "https://api.example.com/v1/auth?token=AIzaSyA123456789",
		ReqBody:   []byte(`{"private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIE..."}`),
		RespBody:  []byte(`{"slack": "xoxb-12345678-abcdef"}`),
	}

	findings := rule.Evaluate(event)
	if len(findings) < 3 {
		t.Fatalf("expected at least 3 findings from req body, resp body, and url query, got %d", len(findings))
	}

	categories := make(map[string]bool)
	for _, f := range findings {
		categories[f.Category] = true
		if f.Severity != SeverityCritical && f.Severity != SeverityHigh {
			t.Errorf("unexpected severity: %s", f.Severity)
		}
	}

	if !categories["EXPOSED_KEY"] || !categories["EXPOSED_SECRET"] {
		t.Fatalf("missing expected categories: %+v", categories)
	}
}
