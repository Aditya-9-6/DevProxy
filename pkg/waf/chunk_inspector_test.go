package waf

import "testing"

func TestChunkInspector(t *testing.T) {
	ci := NewChunkInspector()
	tests := []struct {
		name    string
		line    string
		wantErr bool
	}{
		{"valid", "1a", false},
		{"valid_ext", "1a;extension=val", false},
		{"invalid_size", "zz", true},
		{"smuggling", "1a;Transfer-Encoding=chunked", true},
		{"empty", "", true},
		{"multi_semicolon", "1a;ext1=val;ext2=val", false},
		{"non_ascii", "\x80", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ci.InspectChunkHeader([]byte(tt.line))
			if (err != nil) != tt.wantErr {
				t.Errorf("InspectChunkHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
