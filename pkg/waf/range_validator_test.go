package waf

import (
	"testing"
)

func TestRangeValidator(t *testing.T) {
	tests := []struct {
		name   string
		header string
		valid  bool
	}{
		{"Valid Single", "bytes=0-100", true},
		{"Valid Multiple", "bytes=0-100,101-200", true},
		{"Valid Open Ended", "bytes=0-100,101-", true},
		{"Overlap", "bytes=0-100,50-150", false},
		{"Overlap Open Ended", "bytes=0-,10-", false},
		{"Invalid Start", "bytes=-10-100", false},
		{"Invalid End Before Start", "bytes=100-50", false},
		{"Too Many Parts", "bytes=0-1,2-3,4-5,6-7,8-9,10-11,12-13,14-15,16-17,18-19,20-21,22-23,24-25,26-27,28-29,30-31,32-33", false},
		{"Invalid format", "bytes=0", false},
		{"Empty", "", false},
		{"Empty parts", "bytes=0-100,,200-300", false},
		{"Invalid characters", "bytes=a-b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateRangeHeader(tt.header); got != tt.valid {
				t.Errorf("ValidateRangeHeader() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func BenchmarkRangeValidator(b *testing.B) {
	header := "bytes=0-100,101-200,201-300,301-400,401-500,501-600,601-700,701-800,801-900,901-1000"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateRangeHeader(header)
	}
}
