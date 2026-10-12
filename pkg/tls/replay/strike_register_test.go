package replay

import "testing"

func TestStrikeRegister(t *testing.T) {
	sr := NewStrikeRegister(1024)
	digest := uint64(12345)

	if sr.IsReplay(digest) {
		t.Error("First access should not be replay")
	}
	if !sr.IsReplay(digest) {
		t.Error("Second access should be detected as replay")
	}
	sr.Rotate()
	if sr.IsReplay(digest) {
		t.Error("After rotation, digest should be accepted again")
	}
}
