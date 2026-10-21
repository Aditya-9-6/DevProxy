package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	nodeHealthStatusUnknown uint64 = iota
	nodeHealthStatusHealthy
	nodeHealthStatusUnhealthy
)

// NodeHealthChecker monitors the health of a single upstream node.
type NodeHealthChecker struct {
	url              string
	client           *http.Client
	ring             *HealthRing
	status           atomic.Uint64 // 0=unknown, 1=healthy, 2=unhealthy
	consecFail       atomic.Uint64
	consecSuc        atomic.Uint64
	failureThreshold uint64
	successThreshold uint64
	baseInterval     time.Duration
	maxInterval      time.Duration
	minInterval      time.Duration
	ctx              context.Context
	cancel           context.CancelFunc
	wg               *sync.WaitGroup
}

// NewNodeHealthChecker creates a new NodeHealthChecker.
func NewNodeHealthChecker(ctx context.Context, url string, client *http.Client, failureThreshold, successThreshold uint64, baseInterval time.Duration, wg *sync.WaitGroup) *NodeHealthChecker {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = http.DefaultClient
	}

	ctx, cancel := context.WithCancel(ctx)
	return &NodeHealthChecker{
		url:              url,
		client:           client,
		ring:             NewHealthRing(),
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		baseInterval:     baseInterval,
		maxInterval:      baseInterval * 8,
		minInterval:      baseInterval / 4,
		ctx:              ctx,
		cancel:           cancel,
		wg:               wg,
	}
}

// Start begins the health check loop.
func (n *NodeHealthChecker) Start() {
	if n.wg != nil {
		n.wg.Add(1)
	}
	go n.loop()
}

// Stop stops the health check loop.
func (n *NodeHealthChecker) Stop() {
	n.cancel()
}

// Status returns the current health status (0=unknown, 1=healthy, 2=unhealthy).
func (n *NodeHealthChecker) Status() uint64 {
	return n.status.Load()
}

// Ring returns the health ring for this node.
func (n *NodeHealthChecker) Ring() *HealthRing {
	return n.ring
}

func (n *NodeHealthChecker) loop() {
	if n.wg != nil {
		defer n.wg.Done()
	}

	interval := n.baseInterval
	for {
		if n.ctx.Err() != nil {
			return
		}

		start := time.Now()
		err := n.probe()
		latency := time.Since(start)

		if err == nil {
			interval = n.recordSuccess(interval, latency)
		} else {
			interval = n.recordFailure(interval)
		}

		if !n.waitForNextProbe(interval) {
			return
		}
	}
}

func (n *NodeHealthChecker) recordSuccess(interval time.Duration, latency time.Duration) time.Duration {
	n.ring.Record(true, healthDurationToUint64(latency))
	n.consecSuc.Add(1)
	n.consecFail.Store(0)

	if n.Status() == nodeHealthStatusHealthy {
		interval = increaseHealthCheckInterval(interval, n.maxInterval)
	}
	if n.consecSuc.Load() >= n.successThreshold {
		n.status.Store(nodeHealthStatusHealthy)
	}

	return interval
}

func (n *NodeHealthChecker) recordFailure(interval time.Duration) time.Duration {
	n.ring.Record(false, 0)
	n.consecFail.Add(1)
	n.consecSuc.Store(0)

	interval = decreaseHealthCheckInterval(interval, n.minInterval)
	if n.consecFail.Load() >= n.failureThreshold {
		n.status.Store(nodeHealthStatusUnhealthy)
	}

	return interval
}

func (n *NodeHealthChecker) waitForNextProbe(interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()

	select {
	case <-n.ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (n *NodeHealthChecker) probe() error {
	parentCtx := n.ctx
	if parentCtx == nil {
		parentCtx = context.Background()
	}

	probeCtx, cancel := context.WithTimeout(parentCtx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, n.url, nil)
	if err != nil {
		return err
	}

	client := n.client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	return fmt.Errorf("unexpected status: %d", resp.StatusCode)
}

// healthDurationToUint64 converts a duration's nanosecond count to uint64.
// Negative durations are clamped because time.Duration is a signed type.
func healthDurationToUint64(duration time.Duration) uint64 {
	if duration <= 0 {
		return 0
	}
	// The guard guarantees that the signed value is non-negative and therefore
	// fits within uint64 without losing information.
	return uint64(duration) // #nosec G115
}

func increaseHealthCheckInterval(interval, maximum time.Duration) time.Duration {
	if interval >= maximum {
		return maximum
	}
	if maximum > 0 && interval > maximum/2 {
		return maximum
	}
	return interval * 2
}

func decreaseHealthCheckInterval(interval, minimum time.Duration) time.Duration {
	half := interval / 2
	if half < minimum {
		return minimum
	}
	return half
}

// HealthOrchestrator manages health checking for multiple upstream nodes.
type HealthOrchestrator struct {
	mu               sync.RWMutex
	nodes            map[string]*NodeHealthChecker
	client           *http.Client
	failureThreshold uint64
	successThreshold uint64
	baseInterval     time.Duration
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// NewHealthOrchestrator creates a new HealthOrchestrator.
func NewHealthOrchestrator(failureThreshold, successThreshold uint64, baseInterval time.Duration) *HealthOrchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthOrchestrator{
		nodes:            make(map[string]*NodeHealthChecker),
		client:           &http.Client{Timeout: 5 * time.Second},
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		baseInterval:     baseInterval,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// AddNode adds a node to be monitored.
func (o *HealthOrchestrator) AddNode(id, url string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Checking the context while holding the lock serializes AddNode with
	// Stop, preventing a WaitGroup Add from racing with Wait.
	if o.ctx.Err() != nil {
		return
	}

	if previous, exists := o.nodes[id]; exists {
		previous.Stop()
	}

	checker := NewNodeHealthChecker(o.ctx, url, o.client, o.failureThreshold, o.successThreshold, o.baseInterval, &o.wg)
	o.nodes[id] = checker
	checker.Start()
}

// RemoveNode stops monitoring and removes a node.
func (o *HealthOrchestrator) RemoveNode(id string) {
	o.mu.Lock()
	checker, exists := o.nodes[id]
	if exists {
		delete(o.nodes, id)
	}
	o.mu.Unlock()

	if exists {
		checker.Stop()
	}
}

// Status returns the health status of a node.
func (o *HealthOrchestrator) Status(id string) (uint64, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	checker, ok := o.nodes[id]
	if !ok {
		return nodeHealthStatusUnknown, false
	}
	return checker.Status(), true
}

// Ring returns the health ring for a node.
func (o *HealthOrchestrator) Ring(id string) (*HealthRing, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	checker, ok := o.nodes[id]
	if !ok {
		return nil, false
	}
	return checker.Ring(), true
}

// Stop stops all monitoring and cancels the context.
func (o *HealthOrchestrator) Stop() {
	o.mu.Lock()
	o.cancel()
	o.mu.Unlock()
	o.wg.Wait()
}
