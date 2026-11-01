package http2flow

import (
	"testing"
)

func BenchmarkLedgerConsumeAndReplenish(b *testing.B) {
	ledger := NewLedger(1000000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ledger.Consume(10)
			ledger.Replenish(10)
		}
	})
}

func BenchmarkLeaseAcquireRelease(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l := AcquireLease()
		l.Release()
	}
}
