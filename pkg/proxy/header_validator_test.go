package proxy

import (
	"testing"
)

func TestHeaderValidator(t *testing.T) {
	tests := []struct {
		name    string
		masks   []uint64
		wantErr bool
	}{
		{"Valid", []uint64{MaskContentLength, MaskConnection}, false},
		{"Invalid", []uint64{MaskContentLength, MaskTransferEncoding}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewHeaderValidator()
			var err error
			for _, m := range tt.masks {
				err = v.ObserveHeader(m)
				if err != nil {
					break
				}
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("ObserveHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
