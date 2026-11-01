package quicretry

import (
	"net/netip"
	"testing"
	"time"
)

func BenchmarkTokenGenerationAndValidation(b *testing.B) {
	tm, err := NewTokenManager(5 * time.Minute)
	if err != nil {
		b.Fatal(err)
	}

	ip := netip.MustParseAddr("192.0.2.1")
	dcid := []byte("original-dcid-bytes")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token, err := tm.GenerateToken(ip, dcid)
		if err != nil {
			b.Fatal(err)
		}
		_, err = tm.ValidateToken(ip, token)
		if err != nil {
			b.Fatal(err)
		}
	}
}
