package analysis

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestRadixTrie_Lookup(t *testing.T) {
	trie := DefaultBlocklistTrie()

	// Test Cloud Metadata IP
	entry, found := trie.Lookup(net.ParseIP("169.254.169.254"))
	if !found || entry.Category != "CLOUD_METADATA" {
		t.Fatalf("Failed to match cloud metadata IP: found=%v, entry=%v", found, entry)
	}

	// Test Internal RFC 1918
	entry, found = trie.Lookup(net.ParseIP("10.240.0.5"))
	if !found || entry.Category != "INTERNAL_NETWORK" {
		t.Fatalf("Failed to match 10.0.0.0/8 internal IP: found=%v, entry=%v", found, entry)
	}

	// Test Public Internet IP (should not match)
	_, found = trie.Lookup(net.ParseIP("8.8.8.8"))
	if found {
		t.Fatalf("Public IP 8.8.8.8 should not match blocklist")
	}
}

func TestRulesEngine_SecretDetection(t *testing.T) {
	engine := NewSecurityRulesEngine()

	event := &ringbuffer.TrafficEvent{
		ID:        "test-secret-1",
		Timestamp: time.Now(),
		Host:      "api.example.com",
		Method:    "POST",
		URL:       "https://api.example.com/deploy",
		ReqBody:   []byte(`{"aws_key": "AKIAIOSFODNN7EXAMPLE", "github": "ghp_1234567890abcdefghijklmnopqrstuvwxyz"}`),
	}

	findings := engine.Analyze(event)
	if len(findings) < 2 {
		t.Fatalf("Expected at least 2 secret findings, got %d", len(findings))
	}

	hasAWS := false
	hasGH := false
	for _, f := range findings {
		if f.RuleName == "AWS Access Key ID" {
			hasAWS = true
		}
		if f.RuleName == "GitHub Personal Access Token" {
			hasGH = true
		}
	}

	if !hasAWS || !hasGH {
		t.Fatalf("Missing expected findings: hasAWS=%v, hasGH=%v", hasAWS, hasGH)
	}
}

func TestRulesEngine_InsecureCookie(t *testing.T) {
	engine := NewSecurityRulesEngine()

	respHeaders := make(http.Header)
	respHeaders.Add("Set-Cookie", "session_id=abc123xyz; Path=/") // Missing HttpOnly, Missing Secure, Missing SameSite

	event := &ringbuffer.TrafficEvent{
		ID:          "test-cookie-1",
		Timestamp:   time.Now(),
		Host:        "app.example.com",
		Method:      "GET",
		URL:         "https://app.example.com/login",
		TLS:         true,
		StatusCode:  200,
		RespHeaders: respHeaders,
	}

	findings := engine.Analyze(event)
	if len(findings) == 0 {
		t.Fatalf("Expected cookie findings, got 0")
	}

	hasHttpOnly := false
	hasSecure := false
	for _, f := range findings {
		if f.RuleName == "Missing HttpOnly Flag" {
			hasHttpOnly = true
		}
		if f.RuleName == "Missing Secure Flag" {
			hasSecure = true
		}
	}

	if !hasHttpOnly || !hasSecure {
		t.Fatalf("Expected missing HttpOnly and Secure flags, got: hasHttpOnly=%v, hasSecure=%v", hasHttpOnly, hasSecure)
	}
}

func TestRulesEngine_StackTrace(t *testing.T) {
	engine := NewSecurityRulesEngine()

	event := &ringbuffer.TrafficEvent{
		ID:         "test-trace-1",
		Timestamp:  time.Now(),
		Host:       "api.example.com",
		Method:     "GET",
		URL:        "https://api.example.com/user/42",
		StatusCode: 500,
		RespBody: []byte(`Traceback (most recent call last):
  File "server.py", line 42, in get_user
    raise KeyError("user not found")`),
	}

	findings := engine.Analyze(event)
	if len(findings) == 0 {
		t.Fatalf("Expected stack trace finding, got 0")
	}

	if findings[0].Category != "INFORMATION_DISCLOSURE" {
		t.Fatalf("Expected category INFORMATION_DISCLOSURE, got %s", findings[0].Category)
	}
}
