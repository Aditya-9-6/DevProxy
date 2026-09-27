package analysis

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

// ResultHandler is a callback interface for analyzed events and findings.
type ResultHandler interface {
	HandleResult(event *ringbuffer.TrafficEvent, findings []*Finding)
}

// AnalysisWorkerPool manages background worker threads pulling from the ring buffer.
type AnalysisWorkerPool struct {
	ringBuffer    *ringbuffer.RingBuffer
	engine        *SecurityRulesEngine
	handler       ResultHandler
	concurrency   int
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	totalAnalyzed atomic.Uint64
	totalFindings atomic.Uint64
	totalScanTime atomic.Uint64 // nanoseconds
}

// NewAnalysisWorkerPool creates a new pool of analysis workers.
func NewAnalysisWorkerPool(rb *ringbuffer.RingBuffer, engine *SecurityRulesEngine, handler ResultHandler, workers int) *AnalysisWorkerPool {
	if workers <= 0 {
		workers = 4
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &AnalysisWorkerPool{
		ringBuffer:  rb,
		engine:      engine,
		handler:     handler,
		concurrency: workers,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start spawns the background worker goroutines.
func (p *AnalysisWorkerPool) Start() {
	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go p.workerLoop(i)
	}
}

func (p *AnalysisWorkerPool) workerLoop(workerID int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		default:
		}

		event := p.ringBuffer.Pop()
		if event == nil {
			// Buffer closed
			return
		}

		start := time.Now()
		findings := p.engine.Analyze(event)
		duration := time.Since(start)

		p.totalScanTime.Add(uint64(duration.Nanoseconds()))
		p.totalAnalyzed.Add(1)
		if len(findings) > 0 {
			p.totalFindings.Add(uint64(len(findings)))
		}

		if p.handler != nil {
			p.handler.HandleResult(event, findings)
		}
	}
}

// Stats returns performance metrics of the analysis pool.
func (p *AnalysisWorkerPool) Stats() (analyzed uint64, findings uint64, avgScanMicroseconds float64) {
	a := p.totalAnalyzed.Load()
	f := p.totalFindings.Load()
	t := p.totalScanTime.Load()

	avgUs := 0.0
	if a > 0 {
		avgUs = float64(t) / float64(a) / 1000.0
	}
	return a, f, avgUs
}

// Stop gracefully shuts down the worker pool.
func (p *AnalysisWorkerPool) Stop() {
	p.cancel()
	p.ringBuffer.Close()
	p.wg.Wait()
}
