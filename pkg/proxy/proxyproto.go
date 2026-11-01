package proxy

import (
	"fmt"
	"io"
	"net/netip"
	"github.com/Aditya-9-6/DevProxy/pkg/protocol/proxyproto"
)

type ProxyProtoHandler struct {
	decoder    *proxyproto.Decoder
	trustedCIDR []netip.Prefix
}

func NewProxyProtoHandler(trusted []netip.Prefix) *ProxyProtoHandler {
	return &ProxyProtoHandler{
		decoder:     proxyproto.NewDecoder(),
		trustedCIDR: trusted,
	}
}

func (h *ProxyProtoHandler) IsTrusted(remoteAddr netip.Addr) bool {
	if len(h.trustedCIDR) == 0 {
		return true
	}
	for _, prefix := range h.trustedCIDR {
		if prefix.Contains(remoteAddr) {
			return true
		}
	}
	return false
}

func (h *ProxyProtoHandler) ParseHeader(r io.Reader, remoteAddr netip.Addr) (proxyproto.Header, error) {
	if !h.IsTrusted(remoteAddr) {
		return proxyproto.Header{}, fmt.Errorf("proxyproto: untrusted remote address %s", remoteAddr)
	}
	hdr, _, err := h.decoder.Decode(r)
	return hdr, err
}
