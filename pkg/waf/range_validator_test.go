package waf

import "testing"

func TestRangeValidator(t *testing.T) {
	v := NewRangeValidator()
	tests := []struct {
		name   string
		header string
		want   bool
	}{{"Valid", "bytes=0-100, 200-300", true}, {"Overlap", "bytes=0-100, 50-150", false}, {"TooMany", "bytes=0-1, 2-3, 4-5, 6-7, 8-9, 10-11, 12-13, 14-15, 16-17, 18-19, 20-21, 22-23, 24-25, 26-27, 28-29, 30-31, 32-33", false}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.ValidateRangeHeader(tt.header); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
