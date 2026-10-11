package proxy

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// ConnectionShard manages a pool of workers pinned to specific CPU cores.
type ConnectionShard struct {
	numWorkers int
	nextShard  atomic.Uint64
	queues     []chan func()
	wg         sync.WaitGroup
}

// NewConnectionShard initializes a sharding engine based on available CPUs.
func NewConnectionShard(workers int) *ConnectionShard {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	shard := &ConnectionShard{
		numWorkers: workers,
		queues:     make([]chan func(), workers),
	}
	for i := 0; i < workers; i++ {
		shard.queues[i] = make(chan func(), 1024)
		shard.wg.Add(1)
		go shard.worker(i)
	}
	return shard
}

func (s *ConnectionShard) worker(id int) {
	defer s.wg.Done()
	runtime.LockOSThread()
	for task := range s.queues[id] {
		task()
	}
}

// Dispatch routes a task to a specific core-pinned worker.
func (s *ConnectionShard) Dispatch(task func()) {
	idx := s.nextShard.Add(1) % uint64(s.numWorkers)
	s.queues[idx] <- task
}

// Close shuts down all worker queues.
func (s *ConnectionShard) Close() {
	for _, q := range s.queues {
		close(q)
	}
	s.wg.Wait()
}
