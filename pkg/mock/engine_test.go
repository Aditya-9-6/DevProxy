package mock

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestMapLocalRule(t *testing.T) {
	eng := NewEngine()

	// 1. Inline body rule
	err := eng.AddMapLocal(&MapLocalRule{
		Enabled:     true,
		Pattern:     `^https?://api\.example\.com/users`,
		StatusCode:  http.StatusOK,
		ContentType: "application/json",
		InlineBody:  `{"users": ["alice", "bob"]}`,
	})
	if err != nil {
		t.Fatalf("unexpected error adding rule: %v", err)
	}

	rule, body, err := eng.MatchMapLocal("https://api.example.com/users/list")
	if err != nil {
		t.Fatalf("unexpected error matching: %v", err)
	}
	if rule == nil {
		t.Fatal("expected rule to match, got nil")
	}
	if string(body) != `{"users": ["alice", "bob"]}` {
		t.Fatalf("unexpected body: %s", string(body))
	}

	// 2. File-based rule
	tmpDir := t.TempDir()
	mockFile := filepath.Join(tmpDir, "mock.json")
	if err := os.WriteFile(mockFile, []byte(`{"mocked": true}`), 0600); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}

	err = eng.AddMapLocal(&MapLocalRule{
		Enabled:    true,
		Pattern:    `^https?://api\.example\.com/file`,
		StatusCode: http.StatusCreated,
		FilePath:   mockFile,
	})
	if err != nil {
		t.Fatalf("unexpected error adding file rule: %v", err)
	}

	rule2, body2, err := eng.MatchMapLocal("https://api.example.com/file")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule2 == nil || string(body2) != `{"mocked": true}` {
		t.Fatalf("unexpected body from file: %s", string(body2))
	}

	// 3. Non-matching URL
	rule3, _, _ := eng.MatchMapLocal("https://other.com/data")
	if rule3 != nil {
		t.Fatal("expected nil match for unrelated URL")
	}
}

func TestMapRemoteRule(t *testing.T) {
	eng := NewEngine()

	err := eng.AddMapRemote(&MapRemoteRule{
		Enabled:    true,
		Pattern:    `^https?://api\.production\.com/.*`,
		TargetHost: "localhost:3000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	newURL, newHost, matched := eng.MatchMapRemote("https://api.production.com/v1/items", "api.production.com")
	if !matched {
		t.Fatal("expected match for MapRemote")
	}
	if newHost != "localhost:3000" {
		t.Fatalf("expected host localhost:3000, got %s", newHost)
	}
	if newURL != "https://localhost:3000/v1/items" {
		t.Fatalf("expected url https://localhost:3000/v1/items, got %s", newURL)
	}
}

func TestChaosRule(t *testing.T) {
	eng := NewEngine()

	err := eng.AddChaosRule(&ChaosRule{
		Enabled:     true,
		Pattern:     `^https?://flaky\.api\.com/.*`,
		DelayMs:     5,
		ErrorRate:   1.0, // 100% failure
		ErrorStatus: http.StatusBadGateway,
		ErrorBody:   `{"error": "Simulated Gateway Failure"}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	injected, status, body := eng.ApplyChaos("https://flaky.api.com/checkout")
	if !injected {
		t.Fatal("expected chaos injected")
	}
	if status != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", status)
	}
	if body != `{"error": "Simulated Gateway Failure"}` {
		t.Fatalf("unexpected error body: %s", body)
	}

	// Unmatched URL
	injected2, _, _ := eng.ApplyChaos("https://safe.api.com/checkout")
	if injected2 {
		t.Fatal("did not expect chaos on safe url")
	}
}

func TestDeleteAndClearRules(t *testing.T) {
	eng := NewEngine()

	_ = eng.AddMapLocal(&MapLocalRule{ID: "r1", Enabled: true, Pattern: ".*"})
	_ = eng.AddMapRemote(&MapRemoteRule{ID: "r2", Enabled: true, Pattern: ".*", TargetHost: "localhost:80"})
	_ = eng.AddChaosRule(&ChaosRule{ID: "r3", Enabled: true, Pattern: ".*"})

	summary := eng.GetRulesSummary()
	if len(summary["map_local"].([]*MapLocalRule)) != 1 {
		t.Fatal("expected 1 map local rule")
	}

	deleted := eng.DeleteRule("r2")
	if !deleted {
		t.Fatal("expected r2 to be deleted")
	}

	summary = eng.GetRulesSummary()
	if len(summary["map_remote"].([]*MapRemoteRule)) != 0 {
		t.Fatal("expected 0 map remote rules")
	}

	eng.ClearAll()
	summary = eng.GetRulesSummary()
	if len(summary["map_local"].([]*MapLocalRule)) != 0 || len(summary["chaos_rules"].([]*ChaosRule)) != 0 {
		t.Fatal("expected all rules cleared")
	}
}
