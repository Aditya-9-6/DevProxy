package waf

import (
	"testing"
)

func TestScanner(t *testing.T) {
	patterns := []string{"SELECT", "UNION", "<script>"}
	scanner := NewScanner(patterns)
	data := []byte("SELECT * FROM users UNION <script>")

	found := make(map[string]bool)
	scanner.ScanPayload(data, func(match string) {
		found[match] = true
	})

	for _, p := range patterns {
		if !found[p] {
			t.Errorf("expected pattern %s not found", p)
		}
	}
}

func TestAutomatonOverlapping(t *testing.T) {
	patterns := []string{"abc", "bc"}
	auto := NewAutomaton(patterns)
	data := []byte("abc")

	matches := []string{}
	auto.Scan(data, func(match string) {
		matches = append(matches, match)
	})

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}
