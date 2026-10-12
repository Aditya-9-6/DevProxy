package balancer

import (
	"testing"
)

func TestSelector(t *testing.T) {
	addrs := []string{"127.0.0.1:8080", "127.0.0.1:8081"}
	sel := NewSelector(addrs)

	t1 := sel.Select()
	if t1 == nil || t1.Addr == "" {
		t.Fatal("failed to select target")
	}

	// Simulate load
	t1.Stats.RecordConnection()
	t1.Stats.UpdateLatency(500)

	t2 := sel.Select()
	if t2.Addr == t1.Addr {
		t.Errorf("selector did not pick the less loaded target: %s vs %s", t2.Addr, t1.Addr)
	}
}
