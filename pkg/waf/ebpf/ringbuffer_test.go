package ebpf

import (
	"testing"
)

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		want1    uint32
		want2    uint32
		wantBool bool
	}{
		{
			name:     "valid data",
			data:     []byte{1, 0, 0, 0, 2, 0, 0, 0},
			want1:    1,
			want2:    2,
			wantBool: true,
		},
		{
			name:     "data too short",
			data:     []byte{1, 0, 0, 0},
			want1:    0,
			want2:    0,
			wantBool: false,
		},
		{
			name:     "empty data",
			data:     []byte{},
			want1:    0,
			want2:    0,
			wantBool: false,
		},
		{
			name:     "nil data",
			data:     nil,
			want1:    0,
			want2:    0,
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1, got2, gotBool := ParseEvent(tt.data)
			if got1 != tt.want1 {
				t.Errorf("ParseEvent() got1 = %v, want %v", got1, tt.want1)
			}
			if got2 != tt.want2 {
				t.Errorf("ParseEvent() got2 = %v, want %v", got2, tt.want2)
			}
			if gotBool != tt.wantBool {
				t.Errorf("ParseEvent() gotBool = %v, want %v", gotBool, tt.wantBool)
			}
		})
	}
}

func TestEventPool(t *testing.T) {
	v := EventPool.Get()
	if v == nil {
		t.Fatal("EventPool.Get() returned nil")
	}
	buf, ok := v.([]byte)
	if !ok {
		t.Fatalf("EventPool returned wrong type: %T", v)
	}
	if len(buf) != 256 {
		t.Errorf("EventPool returned buffer of length %d, want 256", len(buf))
	}
	EventPool.Put(v)
}
