package transport

import (
	"github.com/Aditya-9-6/DevProxy/pkg/tls/replay"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

type EarlyDataHandler struct {
	register *replay.StrikeRegister
}

func NewEarlyDataHandler() *EarlyDataHandler {
	return &EarlyDataHandler{
		register: replay.NewStrikeRegister(65536),
	}
}

func (h *EarlyDataHandler) Validate(digest uint64) bool {
	return !h.register.IsReplay(digest)
}
