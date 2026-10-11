package waf

import (
	"context"
	"io"
	"sync/atomic"
)

type StreamInspector struct {
	BlockedCount atomic.Uint64
	Inspector    Inspector
}

func (s *StreamInspector) InspectStream(ctx context.Context, r io.Reader, frameType FrameType) error {
	if s.Inspector == nil {
		s.Inspector = &HeuristicInspector{}
	}

	buf := BufferPool.Get().([]byte)
	defer BufferPool.Put(buf)

	var carryOver []byte

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := r.Read(buf)
			if n > 0 {
				combined := append(carryOver, buf[:n]...)
				if s.Inspector.Inspect(combined) {
					s.BlockedCount.Add(1)
				}
				carryOver = append([]byte(nil), combined[len(combined)/2:]...)
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
