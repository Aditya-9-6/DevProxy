package security

import (
	"testing"
)

func TestStreamReassembler(t *testing.T) {
	r := NewStreamReassembler(0)
	var result []byte
	callback := func(b []byte) { result = append(result, b...) }

	r.Push(0, []byte("hello"), callback)
	r.Push(5, []byte("world"), callback)

	if string(result) != "helloworld" {
		t.Errorf("expected helloworld, got %s", string(result))
	}
}

func TestMaxSegs(t *testing.T) {
	r := NewStreamReassembler(0)
	r.maxSegs = 1
	callback := func(b []byte) {}

	r.Push(10, []byte("data"), callback)
	r.Push(20, []byte("data"), callback)

	if r.segments.Len() > 1 {
		t.Error("maxSegs limit not respected")
	}
}
