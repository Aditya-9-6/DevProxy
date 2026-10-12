package proxy

import (
	"testing"
)

func TestHeaderInterner(t *testing.T) {
	interner := NewHeaderInterner()

	k1 := "Content-Type"
	k2 := "Content-Type"

	val1 := interner.Intern(k1)
	val2 := interner.Intern(k2)

	if val1 != val2 {
		t.Errorf("Expected interned strings to be equal, got %p and %p", &val1, &val2)
	}

	if val1 != "Content-Type" {
		t.Errorf("Expected Content-Type, got %s", val1)
	}
}

func BenchmarkHeaderInterner(b *testing.B) {
	interner := NewHeaderInterner()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			interner.Intern("X-Request-ID")
		}
	})
}
