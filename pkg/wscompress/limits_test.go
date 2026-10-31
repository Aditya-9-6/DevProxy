package wscompress

import (
	"testing"
)

func TestValidatorLimits(t *testing.T) {
	limits := Limits{
		MaxOutputSize:       100,
		MaxExpansionRatio:   5.0,
		MaxWindowBits:       15,
	}

	v := NewValidator(limits)

	// Normal tracking
	if err := v.Track(10, 20); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Exceed expansion ratio: input 10, output 60 (ratio 6.0 > 5.0)
	v2 := NewValidator(limits)
	v2.Track(10, 10)
	if err := v2.Track(0, 50); err == nil {
		t.Fatal("expected expansion ratio error, got nil")
	}

	// Exceed max output size
	v3 := NewValidator(limits)
	if err := v3.Track(10, 150); err == nil {
		t.Fatal("expected output size error, got nil")
	}
}
