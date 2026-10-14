package header

import (
	"hash/fnv"
	"net/http"
	"sort"
	"strings"
)

func ComputeHeaderHash(headers http.Header) uint64 {
	h := fnv.New64a()
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		h.Write([]byte(strings.ToLower(k)))
		vals := headers[k]
		sort.Strings(vals)
		for _, v := range vals {
			h.Write([]byte(v))
		}
	}
	return h.Sum64()
}
