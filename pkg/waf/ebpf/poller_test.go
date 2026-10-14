package ebpf

import (
	"sync"
	"testing"
	"time"
)

func TestPollerLifecycle(t *testing.T) {
	p := NewPoller()
	p.Start()
	time.Sleep(10 * time.Millisecond)
	p.Stop()
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
