package http2frame

import (
	"context"
	"io"
)

// FrameEvent represents a parsed HTTP/2 frame event.
type FrameEvent struct {
	Type    byte
	Flags   byte
	StreamID uint32
	Payload []byte
}

// FrameHandler processes HTTP/2 frames.
type FrameHandler interface {
	HandleFrame(ctx context.Context, ev FrameEvent) error
}

// ParseFrames reads and dispatches frames from an io.Reader.
func ParseFrames(ctx context.Context, r io.Reader, h FrameHandler) error {
	header := make([]byte, 9)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, err := io.ReadFull(r, header)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}

		length := uint32(header[0])<<16 | uint32(header[1])<<8 | uint32(header[2])
		fType := header[3]
		flags := header[4]
		streamID := (uint32(header[5])&0x7f)<<24 | uint32(header[6])<<16 | uint32(header[7])<<8 | uint32(header[8])

		payload := make([]byte, length)
		if length > 0 {
			_, err = io.ReadFull(r, payload)
			if err != nil {
				return err
			}
		}

		ev := FrameEvent{
			Type:     fType,
			Flags:    flags,
			StreamID: streamID,
			Payload:  payload,
		}

		if h != nil {
			if err := h.HandleFrame(ctx, ev); err != nil {
				return err
			}
		}
	}
}
