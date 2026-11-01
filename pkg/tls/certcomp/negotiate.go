package certcomp

import (
	"sync/atomic"
)

// Negotiator tracks and negotiates certificate compression algorithms in a handshake.
type Negotiator struct {
	peerSupported []Algorithm
	localSupported []Algorithm
	negotiatedAlg  atomic.Value // stores *Algorithm
}

func NewNegotiator(local []Algorithm) *Negotiator {
	return &Negotiator{
		localSupported: local,
	}
}

func (n *Negotiator) SetPeerSupported(algs []Algorithm) {
	n.peerSupported = algs
}

func (n *Negotiator) Negotiate(preferred Algorithm) (Algorithm, bool) {
	for _, l := range n.localSupported {
		if l == preferred {
			for _, p := range n.peerSupported {
				if p == preferred {
					n.negotiatedAlg.Store(&preferred)
					return preferred, true
				}
			}
		}
	}

	// Fallback to first mutually supported
	for _, l := range n.localSupported {
		for _, p := range n.peerSupported {
			if l == p {
				n.negotiatedAlg.Store(&l)
				return l, true
			}
		}
	}
	return 0, false
}

func (n *Negotiator) Negotiated() (Algorithm, bool) {
	v := n.negotiatedAlg.Load()
	if v == nil {
		return 0, false
	}
	return *v.(*Algorithm), true
}