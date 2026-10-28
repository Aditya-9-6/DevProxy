package relay

import (
	"context"
	"io"
	"net"
	"sync"
	"time"
)

var bufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

// Config configures duplex relay timeouts and behaviors.
type Config struct {
	IdleTimeout time.Duration
}

// FullDuplexRelay relays traffic between two connections until both finish or context is cancelled.
func FullDuplexRelay(ctx context.Context, clientConn, serverConn net.Conn, cfg Config) error {
	state := NewDuplexState()
	errCh := make(chan error, 2)

	go func() {
		errCh <- copyHalf(ctx, clientConn, serverConn, state, true, cfg.IdleTimeout)
	}()

	go func() {
		errCh <- copyHalf(ctx, serverConn, clientConn, state, false, cfg.IdleTimeout)
	}()

	var firstErr error
	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			if err != nil && firstErr == nil && err != io.EOF {
				firstErr = err
				state.MarkClosed()
				_ = clientConn.Close()
				_ = serverConn.Close()
			}
		case <-ctx.Done():
			state.MarkClosed()
			_ = clientConn.Close()
			_ = serverConn.Close()
			if firstErr == nil {
				firstErr = ctx.Err()
			}
		}
	}

	_ = clientConn.Close()
	_ = serverConn.Close()
	return firstErr
}

func copyHalf(ctx context.Context, src, dst net.Conn, state *DuplexState, c2s bool, idleTimeout time.Duration) error {
	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	buf := *bufPtr

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if idleTimeout > 0 {
			_ = src.SetReadDeadline(time.Now().Add(idleTimeout))
		}

		n, err := src.Read(buf)
		if n > 0 {
			if idleTimeout > 0 {
				_ = dst.SetWriteDeadline(time.Now().Add(idleTimeout))
			}
			if _, wErr := dst.Write(buf[:n]); wErr != nil {
				return wErr
			}
		}

		if err != nil {
			if err == io.EOF {
				if tc, ok := dst.(interface{ CloseWrite() error }); ok {
					_ = tc.CloseWrite()
				}
				state.SetHalfClosed(c2s)
				return nil
			}
			return err
		}
	}
}
