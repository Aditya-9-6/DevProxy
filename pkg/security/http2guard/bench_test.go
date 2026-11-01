package http2guard

import (
	"testing"
	"time"
)

func BenchmarkGuardCheckReset(b *testing.B) {
	cfg := GuardConfig{
		MaxRapidResets: 1000000,
		MaxSettings:    1000000,
		RefillWindow:   time.Hour,
		ShardsCount:    16,
	}
	guard := NewGuard(cfg)
	conn := "bench-conn"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = guard.CheckReset(conn)
	}
}

func BenchmarkGuardCheckSettings(b *testing.B) {
	cfg := GuardConfig{
		MaxRapidResets: 1000000,
		MaxSettings:    1000000,
		RefillWindow:   time.Hour,
		ShardsCount:    16,
	}
	guard := NewGuard(cfg)
	conn := "bench-conn"

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = guard.CheckSettings(conn)
	}
}
