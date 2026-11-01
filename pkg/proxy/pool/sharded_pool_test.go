package pool

import (
	"testing"
)

func TestShardedPool(t *testing.T) {
	p := NewShardedPool(4, func() interface{} {
		return make([]byte, 1024)
	})

	hash := AffinityKey("test-client-1")
	obj := p.Get(hash)
	if obj == nil {
		t.Fatal("expected non-nil object from pool")
	}

	buf, ok := obj.([]byte)
	if !ok {
		t.Fatal("expected []byte object")
	}

	if len(buf) != 1024 {
		t.Fatalf("expected buffer length 1024, got %d", len(buf))
	}

	p.Put(hash, buf)
}
