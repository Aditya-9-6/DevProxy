package http2flow

import (
	"testing"
)

func BenchmarkLedger_AddConnSendWindow(b *testing.B) {
	l := NewLedger(65535, 65535)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.AddConnSendWindow(1)
	}
}
