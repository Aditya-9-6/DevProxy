package proxy

import (
	"net"
	"testing"
)

func TestSubnetClassifier(t *testing.T) {
	sc := NewSubnetClassifier()
	sc.AddTier("10.0.0.0/8", 1)
	sc.AddTier("10.1.0.0/16", 2)

	tests := []struct {
		ip   string
		want int
	}{{"10.1.2.3", 2}, {"10.2.2.3", 1}, {"192.168.1.1", 0}}

	for _, tt := range tests {
		if got := sc.Classify(net.ParseIP(tt.ip)); got != tt.want {
			t.Errorf("Classify(%s) = %d; want %d", tt.ip, got, tt.want)
		}
	}
}
