package dashboard

import (
	"bytes"
	"testing"
)

func TestWSDefragmenter(t *testing.T) {
	tests := []struct {
		name   string
		frames []struct {
			fin    bool
			opcode int
			data   []byte
		}
		wantOp   int
		wantData []byte
		wantErr  bool
	}{
		{
			name: "Single unfragmented text frame",
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{{true, OpText, []byte("hello")}},
			wantOp:   OpText,
			wantData: []byte("hello"),
		},
		{
			name: "Fragmented binary message",
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{
				{false, OpBinary, []byte{1, 2}},
				{false, OpContinuation, []byte{3, 4}},
				{true, OpContinuation, []byte{5}},
			},
			wantOp:   OpBinary,
			wantData: []byte{1, 2, 3, 4, 5},
		},
		{
			name: "Control frame during fragmentation",
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{
				{false, OpText, []byte("part1")},
				{true, OpPing, []byte("ping")},
				{true, OpContinuation, []byte("part2")},
			},
			wantOp:   OpText,
			wantData: []byte("part1part2"),
		},
		{
			name: "Error: Fragment before start",
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{{true, OpContinuation, []byte("bad")}},
			wantErr: true,
		},
		{
			name: "Error: Message too large",
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{{true, OpText, make([]byte, 100)}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewWSDefragmenter(50)
			var lastMsg *WSMessage
			var err error

			for _, f := range tt.frames {
				msg, e := d.ProcessFrame(f.fin, f.opcode, f.data)
				if e != nil {
					err = e
					break
				}
				if msg != nil && msg.Opcode < 8 {
					lastMsg = msg
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessFrame() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && lastMsg != nil {
				if lastMsg.Opcode != tt.wantOp {
					t.Errorf("Opcode = %v, want %v", lastMsg.Opcode, tt.wantOp)
				}
				if !bytes.Equal(lastMsg.Data, tt.wantData) {
					t.Errorf("Data = %v, want %v", lastMsg.Data, tt.wantData)
				}
			}
		})
	}
}
