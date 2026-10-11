package analysis

import (
	"testing"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

func TestSecretScanner(t *testing.T) {
	scanner := NewSecretScanner([]string{"AKIA", "STRIPE_"})
	event := &ringbuffer.TrafficEvent{ReqBody: []byte("test AKIA secret")}
	findings := scanner.Evaluate(event)
	if len(findings) == 0 {
		t.Errorf("Expected finding, got none")
	}
}

func FuzzAhoCorasickMatcher_ScanBytes(f *testing.F) {
	f.Add([]byte("sample AKIA secret string"))
	f.Add([]byte("payload without secrets"))
	f.Add([]byte("random binary \x00\xff\xfe data"))

	scanner := NewSecretScanner([]string{"AKIA", "STRIPE_", "GITHUB_PAT"})
	f.Fuzz(func(t *testing.T, data []byte) {
		event := &ringbuffer.TrafficEvent{ReqBody: data}
		_ = scanner.Evaluate(event)
	})
}

func BenchmarkSecretScanner_Evaluate(b *testing.B) {
	scanner := NewSecretScanner([]string{"AKIA", "STRIPE_", "AIzaSy", "GITHUB_PAT"})
	event := &ringbuffer.TrafficEvent{ReqBody: []byte("GET /api/v1/user HTTP/1.1\r\nAuthorization: Bearer AKIAIOSFODNN7EXAMPLE\r\n\r\n")}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scanner.Evaluate(event)
	}
}
