package websocket

import (
	"io"
	"sync"
	"github.com/Aditya-9-6/DevProxy/pkg/proxy/websocket/frame"
)

type Bridge struct {
	maxPayload int64
	mu         sync.Mutex
	closed     bool
}

func NewBridge(maxPayload int64) *Bridge {
	return &Bridge{
		maxPayload: maxPayload,
	}
}

func (b *Bridge) Pump(client io.ReadWriter, backend io.ReadWriter) error {
	errChan := make(chan error, 2)

	go func() {
		for {
			f, err := frame.ReadFrame(client, b.maxPayload)
			if err != nil {
				errChan <- err
				return
			}
			if err := frame.WriteFrame(backend, f); err != nil {
				errChan <- err
				return
			}
		}
	}()

	go func() {
		for {
			f, err := frame.ReadFrame(backend, b.maxPayload)
			if err != nil {
				errChan <- err
				return
			}
			// Server frames should typically be unmasked per RFC 6455
			f.Masked = false
			if err := frame.WriteFrame(client, f); err != nil {
				errChan <- err
				return
			}
		}
	}()

	return <-errChan
}

func (b *Bridge) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
}
