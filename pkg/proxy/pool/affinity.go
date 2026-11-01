package pool

import (
	"hash/fnv"
)

// AffinityKey calculates a hash for connection affinity routing.
func AffinityKey(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}
