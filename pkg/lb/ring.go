package lb

import (
	"hash/crc32"
	"sort"
	"sync/atomic"
)

// Node represents an upstream target.
type Node struct {
	ID       string
	Addr     string
	Load     atomic.Uint64
	Capacity uint64
}

// Ring implements a consistent hashing ring with bounded-load fallback.
type Ring struct {
	nodes        []*Node
	hashRing     []uint32
	nodeMap      map[uint32]*Node
	virtualNodes int
}

func NewRing(nodes []*Node, virtualNodes int) *Ring {
	r := &Ring{
		nodes:        nodes,
		hashRing:     make([]uint32, 0, len(nodes)*virtualNodes),
		nodeMap:      make(map[uint32]*Node),
		virtualNodes: virtualNodes,
	}
	for _, n := range nodes {
		for i := 0; i < virtualNodes; i++ {
			h := crc32.ChecksumIEEE([]byte(n.ID + string(rune(i))))
			r.hashRing = append(r.hashRing, h)
			r.nodeMap[h] = n
		}
	}
	sort.Slice(r.hashRing, func(i, j int) bool { return r.hashRing[i] < r.hashRing[j] })
	return r
}

func (r *Ring) Get(key string) *Node {
	h := crc32.ChecksumIEEE([]byte(key))
	idx := sort.Search(len(r.hashRing), func(i int) bool { return r.hashRing[i] >= h })
	if idx == len(r.hashRing) {
		idx = 0
	}

	// Bounded-load check: max load factor 1.25x
	for i := 0; i < len(r.hashRing); i++ {
		currIdx := (idx + i) % len(r.hashRing)
		n := r.nodeMap[r.hashRing[currIdx]]
		load := n.Load.Load()
		if load < uint64(float64(n.Capacity)*1.25) {
			return n
		}
	}
	return r.nodeMap[r.hashRing[idx]]
}
