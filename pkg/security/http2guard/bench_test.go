package http2guard_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/security/http2guard"
)

var benchmarkSink atomic.Uint64

func benchmarkGuard(b *testing.B, process *http2guard.ProcessLimiter) *http2guard.Guard {
	b.Helper()

	config := http2guard.DefaultConfig()
	config.RapidReset.Burst = http2guard.MaxBurst
	config.Settings.Burst = http2guard.MaxBurst
	config.Process = process
	guard, err := http2guard.NewGuard(config)
	if err != nil {
		b.Fatalf("NewGuard() error = %v", err)
	}
	return guard
}

func benchmarkStreamID(index int) uint32 {
	return uint32(index%int(http2guard.MaxStreamID)) + 1
}

func BenchmarkGuardAdmitRapid(b *testing.B) {
	guard := benchmarkGuard(b, nil)
	b.ReportAllocs()
	b.ResetTimer()

	for index := 0; index < b.N; index++ {
		verdict := guard.AdmitStreamOpen(benchmarkStreamID(index))
		benchmarkSink.Store(uint64(verdict.Action))
	}
}

func BenchmarkGuardAdmitSettings(b *testing.B) {
	guard := benchmarkGuard(b, nil)
	b.ReportAllocs()
	b.ResetTimer()

	for index := 0; index < b.N; index++ {
		verdict := guard.AdmitSettingsACK()
		benchmarkSink.Store(uint64(verdict.Action))
	}
}

func BenchmarkGuardAdmitWithProcessLimit(b *testing.B) {
	process, err := http2guard.NewProcessLimiter(http2guard.ProcessConfig{
		RapidReset: http2guard.BudgetConfig{Burst: 1 << 30, Window: time.Second},
		Settings:   http2guard.BudgetConfig{Burst: 1 << 30, Window: time.Second},
		Shards:     64,
	})
	if err != nil {
		b.Fatalf("NewProcessLimiter() error = %v", err)
	}
	guard := benchmarkGuard(b, process)
	b.ReportAllocs()
	b.ResetTimer()

	for index := 0; index < b.N; index++ {
		verdict := guard.AdmitStreamReset(benchmarkStreamID(index))
		benchmarkSink.Store(uint64(verdict.Action))
	}
}
