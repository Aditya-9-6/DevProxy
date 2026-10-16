package tls

import (
	"context"
	"sync/atomic"
	"time"
)

// KeyRotator manages the atomic rotation of session keys.
type KeyRotator struct {
	active atomic.Pointer[SessionKeyRing]
	ctx    context.Context
	cancel context.CancelFunc
}

func NewKeyRotator(interval time.Duration) (*KeyRotator, error) {
	initial, err := NewSessionKeyRing()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &KeyRotator{ctx: ctx, cancel: cancel}
	r.active.Store(initial)
	go r.run(interval)
	return r, nil
}

func (r *KeyRotator) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			next, err := NewSessionKeyRing()
			if err == nil {
				r.active.Store(next)
			}
		}
	}
}

func (r *KeyRotator) GetActive() *SessionKeyRing {
	return r.active.Load()
}

func (r *KeyRotator) Stop() {
	r.cancel()
}
