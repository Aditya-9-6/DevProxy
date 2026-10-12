package drain

import (
	"sync"
	"testing"
	"time"
)

func TestManager_Drain(t *testing.T) {
	m := NewManager()

	if !m.Register() {
		t.Error("Expected registration to succeed")
	}

	m.Drain()

	if m.Register() {
		t.Error("Expected registration to fail during drain")
	}

	m.Unregister()
	if m.ActiveCount() != 0 {
		t.Errorf("Expected 0 active connections, got %d", m.ActiveCount())
	}
}

func TestManager_ConcurrentAndPool(t *testing.T) {
	m := NewManager()
	var wg sync.WaitGroup
	routines := 50
	iterations := 100

	for i := 0; i < routines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if m.Register() {
					cs := m.GetState()
					time.Sleep(time.Microsecond)
					m.PutState(cs)
					m.Unregister()
				}
			}
		}()
	}

	// Test pooling directly
	cs := m.GetState()
	if !cs.Active {
		t.Error("Expected pooled connection state to be active")
	}
	m.PutState(cs)

	wg.Wait()

	m.Drain()
	go func() {
		time.Sleep(10 * time.Millisecond)
		m.Unregister()
	}()

	// Should block until active count reaches 0
	m.Wait()
	if m.ActiveCount() != 0 {
		t.Errorf("Expected active count to be 0 after Wait, got %d", m.ActiveCount())
	}
}
