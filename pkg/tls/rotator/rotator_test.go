package rotator

import (
	"testing"
)

func TestRotator(t *testing.T) {
	r := NewRotator()
	key1 := r.Get()
	if key1 == nil {
		t.Fatal("expected initial key")
	}

	r.Rotate()
	key2 := r.Get()
	if key2 == nil || key1 == key2 {
		t.Fatal("expected new key after rotation")
	}
}
