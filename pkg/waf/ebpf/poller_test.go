package ebpf

import (
	"sync"
	"testing"
	"time"
)

func TestPollerLifecycle(t *testing.T) {
	p := NewPoller()
	p.Start()
	p.Start() // Test idempotency
	time.Sleep(10 * time.Millisecond)
	p.Stop()
}

func TestPollerBackpressure(t *testing.T) {
	p := NewPoller()
	// Fill the channel to trigger backpressure
	for i := 0; i < 1024; i++ {
		p.Events <- make([]byte, 256)
	}
	p.Start()
	time.Sleep(50 * time.Millisecond)
	p.Stop()

	drops, _ := p.Telemetry.GetStats()
	if drops == 0 {
		t.Error("expected drops due to backpressure, got 0")
	}
}

func TestPollerConcurrency(t *testing.T) {
	p := NewPoller()
	p.Start()
	defer p.Stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			select {
			case data := <-p.Events:
				EventPool.Put(data)
			case <-time.After(1 * time.Second):
				return
			}
		}
	}()
	wg.Wait()
}

func TestTelemetryCounters(t *testing.T) {
	tele := &Telemetry{}
	tele.RecordDrop()
	tele.RecordHit()
	drops, hits := tele.GetStats()
	if drops != 1 || hits != 1 {
		t.Errorf("expected 1/1, got %d/%d", drops, hits)
	}
}

func BenchmarkEventProcessing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := EventPool.Get().([]byte)
		_, _, _ = ParseEvent(buf)
		EventPool.Put(buf)
	}
}
