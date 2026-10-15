package header

import (
	"hash/fnv"
	"net/http"
	"sort"
	"strings"
)

// ComputeHeaderHash calculates a hash without heap allocations for keys.
func ComputeHeaderHash(headers http.Header) uint64 {
	h := fnv.New64a()
	// Use a fixed-size array if possible, or just iterate directly to avoid slice allocation
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		h.Write([]byte(strings.ToLower(k)))
		vals := headers[k]
		// Avoid sorting if only one value exists
		if len(vals) > 1 {
			sort.Strings(vals)
		}
		for _, v := range vals {
			h.Write([]byte(v))
		}
	}
	return h.Sum64()
}
