package proxy

import (
	"testing"
)

func TestAtomicDictionary(t *testing.T) {
	dict := NewAtomicDictionary()
	dict.Put("Content-Type", "application/json")
	dict.Put("X-Request-ID", "abc-123")

	val, ok := dict.Get("Content-Type")
	if !ok || val != "application/json" {
		t.Errorf("Expected application/json, got %s", val)
	}

	_, ok = dict.Get("Non-Existent")
	if ok {
		t.Error("Expected false for non-existent key")
	}
}

func BenchmarkAtomicDictionary_Get(b *testing.B) {
	dict := NewAtomicDictionary()
	dict.Put("k", "v")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dict.Get("k")
	}
}
