package splice

import (
	"fmt"
	"syscall"
)

func NewPool(size int) (*Pool, error) {
	if size > 64 {
		size = 64
	}
	pipes := make([]PipePair, size)
	for i := 0; i < size; i++ {
		var fds [2]int
		if err := syscall.Pipe(fds[:]); err != nil {
			return nil, fmt.Errorf("failed to create pipe: %w", err)
		}
		pipes[i] = PipePair{Reader: fds[0], Writer: fds[1]}
	}
	return &Pool{pipes: pipes, capacity: size}, nil
}

func (p *Pool) Acquire() (PipePair, bool) {
	for {
		mask := p.mask.Load()
		for i := 0; i < p.capacity; i++ {
			if (mask & (1 << i)) == 0 {
				if p.mask.CompareAndSwap(mask, mask|(1<<i)) {
					return p.pipes[i], true
				}
				break
			}
		}
		if mask == (1<<p.capacity)-1 {
			return PipePair{}, false
		}
	}
}

func (p *Pool) Release(pipe PipePair) {
	for i := 0; i < p.capacity; i++ {
		if p.pipes[i] == pipe {
			for {
				mask := p.mask.Load()
				if p.mask.CompareAndSwap(mask, mask&^(1<<i)) {
					return
				}
			}
		}
	}
}
