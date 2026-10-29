package goaway

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type ConnectionState int32

const (
	StateActive ConnectionState = iota
	StateDraining
	StateTerminated
)

var (
	ErrConnectionDraining   = errors.New("connection is draining; new streams prohibited")
	ErrConnectionTerminated = errors.New("connection is terminated")
	ErrInvalidLastStreamID  = errors.New("last stream ID must be non-increasing")
)

// Coordinator manages connection lifecycle states and GOAWAY frame enforcement.
type Coordinator struct {
	state          int32 // ConnectionState
	lastStreamID   uint32
	mu             sync.Mutex
	graceDeadline  time.Time
}

func NewCoordinator() *Coordinator {
	return &Coordinator{
		state:        int32(StateActive),
		lastStreamID: 0xFFFFFFFF,
	}
}

func (c *Coordinator) State() ConnectionState {
	return ConnectionState(atomic.LoadInt32(&c.state))
}

func (c *Coordinator) CanCreateStream(streamID uint32) error {
	st := c.State()
	if st == StateTerminated {
		return ErrConnectionTerminated
	}
	if st == StateDraining {
		c.mu.Lock()
		defer c.mu.Unlock()
		if streamID > c.lastStreamID {
			return ErrConnectionDraining
		}
	}
	return nil
}

func (c *Coordinator) HandleGoAway(f *Frame, gracePeriod time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if f.LastStreamID > c.lastStreamID && c.State() == StateDraining {
		return ErrInvalidLastStreamID
	}

	c.lastStreamID = f.LastStreamID

	if atomic.LoadInt32(&c.state) == int32(StateActive) {
		atomic.StoreInt32(&c.state, int32(StateDraining))
		c.graceDeadline = time.Now().Add(gracePeriod)
	}
	return nil
}

func (c *Coordinator) Terminate() {
	atomic.StoreInt32(&c.state, int32(StateTerminated))
}
