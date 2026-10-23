package upstream

import (
	"context"
	"errors"
	"github.com/gossipmesh/devproxy/pkg/http2/goaway"
)

var ErrUpstreamDraining = errors.New("upstream proxy connection is draining")

// UpstreamHandler wraps connection coordination for upstream proxy requests.
type UpstreamHandler struct {
	Coordinator *goaway.Coordinator
}

func NewUpstreamHandler(coord *goaway.Coordinator) *UpstreamHandler {
	return &UpstreamHandler{Coordinator: coord}
}

func (h *UpstreamHandler) CheckRequest(ctx context.Context, streamID uint32) error {
	if err := h.Coordinator.CanCreateStream(streamID); err != nil {
		return err
	}
	return nil
}
