package waf

import (
	"hash/fnv"
	"testing"
)

func TestHLLCardinality(t *testing.T) {
	hll := NewHLL(10)
	items := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "1.1.1.1"}

	for _, item := range items {
		h := fnv.New64a()
		h.Write([]byte(item))
		hll.Add(h.Sum64())
	}

	count := hll.Count()
	if count < 2 || count > 4 {
		t.Errorf("Expected cardinality around 3, got %d", count)
	}
}
