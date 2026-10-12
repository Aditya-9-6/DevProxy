package stream

import (
	"testing"
)

func TestJitterBuffer_Resequencing(t *testing.T) {
	jb := NewJitterBuffer()

	// Push out of order
	jb.Push(1, []byte("data1"))
	jb.Push(0, []byte("data0"))

	// Pop in order
	p0 := jb.Pop()
	if p0 == nil || string(p0.Data) != "data0" {
		t.Errorf("Expected data0, got %v", p0)
	}
	jb.Recycle(p0)

	p1 := jb.Pop()
	if p1 == nil || string(p1.Data) != "data1" {
		t.Errorf("Expected data1, got %v", p1)
	}
	jb.Recycle(p1)
}

func TestJitterBuffer_Empty(t *testing.T) {
	jb := NewJitterBuffer()
	if p := jb.Pop(); p != nil {
		t.Error("Expected nil on empty buffer")
	}
}
