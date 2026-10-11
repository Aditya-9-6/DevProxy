package dashboard

import (
	"bytes"
	"errors"
	"testing"
)

func TestWSDefragmenter_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		maxSize int
		frames  []struct {
			fin    bool
			opcode int
			data   []byte
		}
		wantOp   int
		wantData []byte
		wantErr  error
	}{
		{
			name:    "Single unfragmented text frame",
			maxSize: 1024,
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{
				{true, OpText, []byte("hello world")},
			},
			wantOp:   OpText,
			wantData: []byte("hello world"),
		},
		{
			name:    "Fragmented binary payload across 3 frames",
			maxSize: 1024,
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{
				{false, OpBinary, []byte{1, 2, 3}},
				{false, OpContinuation, []byte{4, 5}},
				{true, OpContinuation, []byte{6, 7, 8}},
			},
			wantOp:   OpBinary,
			wantData: []byte{1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			name:    "Interleaved ping control frame during fragmented message",
			maxSize: 1024,
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{
				{false, OpText, []byte("first chunk ")},
				{true, OpPing, []byte("heartbeat")},
				{true, OpContinuation, []byte("second chunk")},
			},
			wantOp:   OpText,
			wantData: []byte("first chunk second chunk"),
		},
		{
			name:    "Error on continuation before start frame",
			maxSize: 1024,
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{
				{false, OpContinuation, []byte("orphan")},
			},
			wantErr: ErrFragmentBeforeStart,
		},
		{
			name:    "Error on nested start frame without FIN",
			maxSize: 1024,
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{
				{false, OpText, []byte("msg1 start")},
				{false, OpText, []byte("msg2 start")},
			},
			wantErr: ErrNestedStartFrame,
		},
		{
			name:    "Error on payload exceeding maxSize",
			maxSize: 10,
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{
				{false, OpBinary, []byte("123456")},
				{true, OpContinuation, []byte("7890exceed")},
			},
			wantErr: ErrMessageTooLarge,
		},
		{
			name:    "Error on fragmented control frame (FIN=false)",
			maxSize: 1024,
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{
				{false, OpPing, []byte("fragmented-ping")},
			},
			wantErr: ErrControlFragmented,
		},
		{
			name:    "Error on invalid UTF-8 in completed text frame",
			maxSize: 1024,
			frames: []struct {
				fin    bool
				opcode int
				data   []byte
			}{
				{true, OpText, []byte{0xff, 0xfe, 0xfd}},
			},
			wantErr: ErrInvalidUTF8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewWSDefragmenter(tt.maxSize)
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

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if lastMsg == nil {
				t.Fatalf("expected completed message, got nil")
			}

			if lastMsg.Opcode != tt.wantOp {
				t.Errorf("opcode mismatch: expected %d, got %d", tt.wantOp, lastMsg.Opcode)
			}

			if !bytes.Equal(lastMsg.Data, tt.wantData) {
				t.Errorf("data mismatch: expected %v, got %v", tt.wantData, lastMsg.Data)
			}
		})
	}
}

func TestWSDefragmenter_Reset(t *testing.T) {
	d := NewWSDefragmenter(1024)
	_, _ = d.ProcessFrame(false, OpText, []byte("partial"))
	d.Reset()

	// After reset, starting a new message should succeed without ErrNestedStartFrame
	msg, err := d.ProcessFrame(true, OpText, []byte("fresh message"))
	if err != nil {
		t.Fatalf("unexpected error after reset: %v", err)
	}
	if string(msg.Data) != "fresh message" {
		t.Fatalf("expected 'fresh message', got %s", string(msg.Data))
	}
}
