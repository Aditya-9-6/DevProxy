package splice

import (
	"fmt"
	"runtime"
)

func NewPool(size int) (*Pool, error) {
	if size > 64 {
		size = 64
	}
	pipes := make([]PipePair, size)
	for i := 0; i < size; i++ {
		r, w, err := createPipe()
		if err != nil {
			return nil, fmt.Errorf("failed to create pipe: %w", err)
		}
		pipes[i] = PipePair{Reader: r, Writer: w, Index: i}
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
		runtime.Gosched()
	}
}

func (p *Pool) Release(pipe PipePair) {
	for {
		mask := p.mask.Load()
		if p.mask.CompareAndSwap(mask, mask&^(1<<pipe.Index)) {
			return
		}
	}
}

func (p *Pool) Close() error {
	for _, pipe := range p.pipes {
		closePipe(pipe.Reader, pipe.Writer)
	}
	return nil
}
