package analysis

import (
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestCustomRules_YAMLLoading(t *testing.T) {
	yamlContent := []byte(`
rules:
  - name: "Internal Corp Secret"
    category: "EXPOSED_SECRET"
    severity: "CRITICAL"
    regex: "corp_sec_[0-9a-zA-Z]{16}"
    description: "Detected internal corp secret"
    remediation: "Revoke key in Vault"
    location: "all"
  - name: "Internal Domain Access"
    category: "RESTRICTED_ACCESS"
    severity: "MEDIUM"
    regex: ".*\\.internal\\.net"
    description: "Hit internal corporate domain"
    remediation: "Mock internal dependencies"
    location: "url"
`)

	rules, err := LoadCustomRulesFromYAML(yamlContent)
	if err != nil {
		t.Fatalf("Failed to parse custom rules YAML: %v", err)
	}

	if len(rules) != 2 {
		t.Fatalf("Expected 2 custom rules, got %d", len(rules))
	}

	engine := NewSecurityRulesEngine()
	engine.AddRules(rules)

	// Test secret detection in body
	event1 := &ringbuffer.TrafficEvent{
		ID:        "cust-1",
		Timestamp: time.Now(),
		URL:       "https://api.github.com/v1",
		ReqBody:   []byte(`{"auth": "corp_sec_abcdef1234567890"}`),
	}
	findings1 := engine.Analyze(event1)
	hasCorpSec := false
	for _, f := range findings1 {
		if f.RuleName == "Internal Corp Secret" {
			hasCorpSec = true
			if f.Severity != SeverityCritical {
				t.Errorf("Expected severity CRITICAL, got %s", f.Severity)
			}
		}
	}
	if !hasCorpSec {
		t.Fatalf("Custom rule 'Internal Corp Secret' failed to trigger on request body")
	}

	// Test URL rule
	event2 := &ringbuffer.TrafficEvent{
		ID:        "cust-2",
		Timestamp: time.Now(),
		URL:       "https://auth.internal.net/oauth/token",
	}
	findings2 := engine.Analyze(event2)
	hasInternalDomain := false
	for _, f := range findings2 {
		if f.RuleName == "Internal Domain Access" {
			hasInternalDomain = true
		}
	}
	if !hasInternalDomain {
		t.Fatalf("Custom rule 'Internal Domain Access' failed to trigger on URL")
	}
}
