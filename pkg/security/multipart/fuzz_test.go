package multipart

import (
	"testing"
)

func FuzzBoundaryMatcher(f *testing.F) {
	f.Add("abc123xyz", "--abc123xyz\r\n")
	f.Fuzz(func(t *testing.T, boundary string, payload string) {
		if len(boundary) == 0 {
			return
		}
		bm := NewBoundaryMatcher(boundary)
		_ = bm.MatchFull([]byte(payload))
		_ = bm.MatchClose([]byte(payload))
	})
}
