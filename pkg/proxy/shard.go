package proxy

import (
	"context"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
)

// ConnectionShard manages a pool of workers pinned to specific CPU cores.
type ConnectionShard struct {
	numWorkers int
	nextShard  atomic.Uint64
	queues     []chan func()
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewConnectionShard initializes a sharding engine based on available CPUs.
func NewConnectionShard(workers int) *ConnectionShard {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	ctx, cancel := context.WithCancel(context.Background())
	shard := &ConnectionShard{
		numWorkers: workers,
		queues:     make([]chan func(), workers),
		ctx:        ctx,
		cancel:     cancel,
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

	q := s.queues[id]
	for {
		select {
		case <-s.ctx.Done():
			// Drain remaining tasks on shutdown
			for {
				select {
				case task, ok := <-q:
					if !ok {
						return
					}
					s.safeExecute(task)
				default:
					return
				}
			}
		case task, ok := <-q:
			if !ok {
				return
			}
			s.safeExecute(task)
		}
	}
}

func (s *ConnectionShard) safeExecute(task func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in ConnectionShard worker: %v", r)
		}
	}()
	task()
}

// Dispatch routes a task to a specific core-pinned worker.
func (s *ConnectionShard) Dispatch(task func()) bool {
	idx := s.nextShard.Add(1) % uint64(s.numWorkers)
	select {
	case <-s.ctx.Done():
		return false
	case s.queues[idx] <- task:
		return true
	}
}

// Close shuts down all worker queues gracefully.
func (s *ConnectionShard) Close() {
	s.cancel()
	for _, q := range s.queues {
		close(q)
	}
	s.wg.Wait()
}
