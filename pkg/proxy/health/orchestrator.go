package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
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
		select {
		case <-n.ctx.Done():
			return
		default:
		}

		start := time.Now()
		err := n.probe()
		latency := time.Since(start)

		if err == nil {
			n.ring.Record(true, uint64(latency.Nanoseconds()))
			n.consecSuc.Add(1)
			n.consecFail.Store(0)

			// Increase interval if healthy
			if n.Status() == 1 {
				interval *= 2
				if interval > n.maxInterval {
					interval = n.maxInterval
				}
			}

			// Check if we can transition to healthy
			if n.consecSuc.Load() >= n.successThreshold {
				n.status.Store(1)
			}
		} else {
			n.ring.Record(false, 0)
			n.consecFail.Add(1)
			n.consecSuc.Store(0)

			// Decrease interval if unhealthy
			interval /= 2
			if interval < n.minInterval {
				interval = n.minInterval
			}

			// Check if we should transition to unhealthy
			if n.consecFail.Load() >= n.failureThreshold {
				n.status.Store(2)
			}
		}

		select {
		case <-n.ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func (n *NodeHealthChecker) probe() error {
	ctx, cancel := context.WithTimeout(n.ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.url, nil)
	if err != nil {
		return err
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("unexpected status: %d", resp.StatusCode)
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

	checker := NewNodeHealthChecker(o.ctx, url, o.client, o.failureThreshold, o.successThreshold, o.baseInterval, &o.wg)
	o.nodes[id] = checker
	checker.Start()
}

// RemoveNode stops monitoring and removes a node.
func (o *HealthOrchestrator) RemoveNode(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if checker, ok := o.nodes[id]; ok {
		checker.Stop()
		delete(o.nodes, id)
	}
}

// Status returns the health status of a node.
func (o *HealthOrchestrator) Status(id string) (uint64, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	checker, ok := o.nodes[id]
	if !ok {
		return 0, false
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
	o.cancel()
	o.wg.Wait()
}
