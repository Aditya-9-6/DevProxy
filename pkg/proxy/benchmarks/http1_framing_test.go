package benchmarks

import (
	"bufio"
	"bytes"
	"testing"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy"
	"github.com/Aditya-9-6/DevProxy/pkg/security"
)

func BenchmarkHTTP1FramingAdapter(b *testing.B) {
	policy := security.NewSmugglingPolicy()
	adapter := proxy.NewHTTP1FramingAdapter(policy)
	raw := []byte("GET /bench HTTP/1.1\r\nHost: localhost\r\nUser-Agent: DevProxyBench\r\n\r\n")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bufio.NewReader(bytes.NewReader(raw))
		_, err := adapter.AdaptRequest(r)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
