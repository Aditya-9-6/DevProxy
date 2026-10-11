package waf

import (
	"bytes"
	"testing"
)

func TestScanner(t *testing.T) {
	patterns := []string{"SELECT", "UNION", "<script>"}
	scanner, _ := NewScanner(patterns)
	data := []byte("SELECT * FROM users UNION <script>")

	found := make(map[string]bool)
	scanner.ScanPayload(bytes.NewReader(data), func(match string) {
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
	auto, _ := NewAutomaton(patterns)
	data := []byte("abc")

	matches := []string{}
	auto.Scan(data, func(match string) {
		matches = append(matches, match)
	})

	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
}

func BenchmarkScan(b *testing.B) {
	scanner, _ := NewScanner([]string{"malicious", "payload", "test"})
	data := []byte("this is a test payload for malicious content")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner.ScanPayload(bytes.NewReader(data), func(m string) {})
	}
}
