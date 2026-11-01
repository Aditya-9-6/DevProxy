package http3

import (
	"net"
	quicretry "github.com/Aditya-9-6/DevProxy/pkg/security/quic/retry"
)

type RetryHandler struct {
	Enabled bool
}

func NewRetryHandler(enabled bool) *RetryHandler {
	return &RetryHandler{
		Enabled: enabled,
	}
}

func (h *RetryHandler) HandleInitial(remoteAddr net.Addr, clientInitial []byte) ([]byte, bool) {
	if !h.Enabled {
		return nil, false
	}
	odcid := []byte("dummy-odcid")
	token, err := quicretry.GenerateToken(remoteAddr.String(), odcid, 0)
	if err != nil {
		return nil, false
	}
	return token, true
}
