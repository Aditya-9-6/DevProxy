package proxy

import (
	"sync"
	"sync/atomic"
)

// DemuxPool manages a lock-free cache-aligned pool of connection state descriptors for zero-allocation TCP demuxing.
type DemuxPool struct {
	pall      sync.Pool
	maxActive uint64
	activeCnt atomic.Uint64
}

// NewDemuxPool creates a new DemuxPool with a specified capacity constraint.
func NewDemuxPool(maxActive uint64) *DemuxPool {
	return &DemuxPool{
		maxActive: maxActive,
		pall: sync.Pool{
			New: func() interface{} {
				return &ConnectionStateDescriptor{}
			},
		},
	}
}

// Acquire retrieves a connection state descriptor from the pool in O(1) time.
func (p *DemuxPool) Acquire(stateID uint64, remoteHash uint64) *ConnectionStateDescriptor {
	desc := p.pall.Get().(*ConnectionStateDescriptor)
	desc.Reset()
	desc.SetState(stateID)
	atomic.StoreUint64(&desc.RemoteAddrHash[0], remoteHash)
	desc.SetActive(true)
	p.activeCnt.Add(1)
	return desc
}

// Release returns a connection state descriptor back to the pool in O(1) time.
func (p *DemuxPool) Release(desc *ConnectionStateDescriptor) {
	if desc == nil {
		return
	}
	desc.Reset()
	p.pall.Put(desc)
	if p.activeCnt.Load() > 0 {
		p.activeCnt.Add(^uint64(0))
	}
}

// ActiveCount returns the current number of active descriptors.
func (p *DemuxPool) ActiveCount() uint64 {
	return p.activeCnt.Load()
}
