package waf

import (
	"context"
	"io"
	"sync/atomic"
)

type StreamInspector struct {
	BlockedCount atomic.Uint64
}

func (s *StreamInspector) InspectStream(ctx context.Context, r io.Reader, frameType FrameType) error {
	buf := BufferPool.Get().([]byte)
	defer BufferPool.Put(buf)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := r.Read(buf)
			if n > 0 {
				if InspectFrame(buf[:n], frameType) {
					s.BlockedCount.Add(1)
				}
			}
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
		}
	}
}
