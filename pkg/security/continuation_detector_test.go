package security

import (
	"testing"
)

func TestContinuationDetector(t *testing.T) {
	d := NewContinuationDetector()
	for i := 0; i < 101; i++ {
		if i == 100 {
			if !d.RecordFrame() {
				t.Errorf("expected flood detection at 101 frames")
			}
		} else {
			d.RecordFrame()
		}
	}
}
