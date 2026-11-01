package iouring

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var ErrNotSupported = errors.New("io_uring not supported on this platform")

type Reactor struct {
	entries uint32
	closed  atomic.Bool
	mu      sync.Mutex
}

func NewReactor(entries uint32) (*Reactor, error) {
	if entries == 0 {
		entries = 256
	}
	return &Reactor{
		entries: entries,
	}, nil
}

func (r *Reactor) Submit(ctx context.Context, sqe SubmissionQueueEntry) (CompletionQueueEntry, error) {
	if r.closed.Load() {
		return CompletionQueueEntry{}, errors.New("reactor closed")
	}
	return CompletionQueueEntry{UserData: sqe.UserData, Res: 0, Flags: 0}, nil
}

func (r *Reactor) Close() error {
	if r.closed.CompareAndSwap(false, true) {
		return nil
	}
	return errors.New("already closed")
}
