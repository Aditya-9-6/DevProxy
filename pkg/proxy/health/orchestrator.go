package health

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Target represents a backend upstream instance to health check.
type Target struct {
	ID       string
	URL      string
	Metadata map[string]string
}

// Config configures the health checking behavior.
type Config struct {
	Interval            time.Duration
	Timeout             time.Duration
	HealthyThreshold    int
	UnhealthyThreshold  int
	Path                string
	ExpectedStatusCodes []int
}

// DefaultConfig returns sensible defaults for health checking.
func DefaultConfig() Config {
	return Config{
		Interval:           5 * time.Second,
		Timeout:            2 * time.Second,
		HealthyThreshold:   2,
		UnhealthyThreshold: 3,
		Path:               "/healthz",
		ExpectedStatusCodes: []int{http.StatusOK},
	}
}

// Orchestrator manages periodic health checks across multiple upstream targets.
type Orchestrator struct {
	config   Config
	client   *http.Client
	targets  map[string]*Target
	buffers  map[string]*RingBuffer
	statuses map[string]*atomic.Value // stores HealthStatus
	mu       sync.RWMutex
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewOrchestrator instantiates a new Health Orchestrator.
func NewOrchestrator(cfg Config) *Orchestrator {
	client := &http.Client{
		Timeout: cfg.Timeout,
	}
	return &Orchestrator{
		config:   cfg,
		client:   client,
		targets:  make(map[string]*Target),
		buffers:  make(map[string]*RingBuffer),
		statuses: make(map[string]*atomic.Value),
		stopChan: make(chan struct{}),
	}
}

// AddTarget registers a new target for health checking.
func (o *Orchestrator) AddTarget(t Target) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.targets[t.ID] = &t
	o.buffers[t.ID] = NewRingBuffer(50)

	val := &atomic.Value{}
	val.Store(StatusUnknown)
	o.statuses[t.ID] = val
}

// RemoveTarget unregisters a target.
func (o *Orchestrator) RemoveTarget(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()

delete(o.targets, id)
	delete(o.buffers, id)
	delete(o.statuses, id)
}

// Start begins background health check loops.
func (o *Orchestrator) Start(ctx context.Context) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	for id, target := range o.targets {
		o.wg.Add(1)
		go func(t *Target, tid string) {
			defer o.wg.Done()
			o.runLoop(ctx, t, tid)
		}(target, id)
	}
}

// Stop halts all background health checks.
func (o *Orchestrator) Stop() {
	close(o.stopChan)
	o.wg.Wait()
}

// GetStatus returns the current health status of a specific target.
func (o *Orchestrator) GetStatus(id string) HealthStatus {
	o.mu.RLock()
	val, exists := o.statuses[id]
	o.mu.RUnlock()

	if !exists {
		return StatusUnknown
	}

	return val.Load().(HealthStatus)
}

func (o *Orchestrator) runLoop(ctx context.Context, target *Target, id string) {
	ticker := time.NewTicker(o.config.Interval)
	defer ticker.Stop()

	// Perform an immediate initial check
	o.checkTarget(ctx, target, id)

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.stopChan:
			return
		case <-ticker.C:
			o.checkTarget(ctx, target, id)
		}
	}
}

func (o *Orchestrator) checkTarget(ctx context.Context, target *Target, id string) {
	checkCtx, cancel := context.WithTimeout(ctx, o.config.Timeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, target.URL+o.config.Path, nil)
	if err != nil {
		o.recordResult(id, CheckResult{
			Timestamp: start,
			Duration:  time.Since(start),
			Status:    StatusUnhealthy,
			Err:       err,
		})
		return
	}

	resp, err := o.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		o.recordResult(id, CheckResult{
			Timestamp: start,
			Duration:  duration,
			Status:    StatusUnhealthy,
			Err:       err,
		})
		return
	}
	defer resp.Body.Close()

	status := StatusUnhealthy
	for _, code := range o.config.ExpectedStatusCodes {
		if resp.StatusCode == code {
			status = StatusHealthy
			break
		}
	}

	var checkErr error
	if status == StatusUnhealthy {
		checkErr = errors.New("unexpected status code: " + http.StatusText(resp.StatusCode))
	}

	o.recordResult(id, CheckResult{
		Timestamp: start,
		Duration:  duration,
		Status:    status,
		Err:       checkErr,
	})
}

func (o *Orchestrator) recordResult(id string, res CheckResult) {
	o.mu.RLock()
	buf, bufExists := o.buffers[id]
	statVal, statExists := o.statuses[id]
	o.mu.RUnlock()

	if !bufExists || !statExists {
		return
	}

	buf.Push(res)
	statVal.Store(res.Status)
}
