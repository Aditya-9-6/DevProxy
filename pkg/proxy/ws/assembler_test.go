package ws

import "testing"

func TestAssembler(t *testing.T) {
	a := NewAssembler()
	f1 := &Frame{Fin: false, Payload: []byte("hello ")}
	f2 := &Frame{Fin: true, Payload: []byte("world")}
	_ = a.Add(f1)
	res := a.Add(f2)
	if string(res) != "hello world" {
		t.Errorf("expected hello world, got %s", string(res))
	}
}
