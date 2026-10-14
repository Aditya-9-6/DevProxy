package websocket

import "testing"

func TestFrameValidator(t *testing.T) {
	t.Parallel()
	v := &FrameValidator{}
	if !v.ValidateMask([4]byte{1, 2, 3, 4}) {
		t.Error("Expected valid mask")
	}
	if v.ValidateMask([4]byte{0, 0, 0, 0}) {
		t.Error("Expected invalid mask")
	}
	if v.ViolationCount.Load() != 1 {
		t.Errorf("Expected 1 violation, got %d", v.ViolationCount.Load())
	}

	v.TrackEntropy([]byte("hello world"))
	if v.EntropyScore.Load() == 0 {
		t.Error("Expected non-zero entropy score")
	}
}

func BenchmarkTrackEntropy(b *testing.B) {
	v := &FrameValidator{}
	data := []byte("some random payload data for entropy calculation")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.TrackEntropy(data)
	}
}
