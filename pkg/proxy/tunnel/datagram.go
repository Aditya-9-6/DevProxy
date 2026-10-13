package tunnel

import (
	"encoding/binary"
	"errors"
	"io"
)

// Datagram represents an RFC 9298 UDP datagram encapsulated over HTTP CONNECT.
type Datagram struct {
	ContextID uint64
	Payload   []byte
}

// WriteDatagram writes an RFC 9298 datagram frame to an io.Writer using variable-length integer encoding.
func WriteDatagram(w io.Writer, contextID uint64, payload []byte) error {
	var ctxBuf [10]byte
	n := binary.PutUvarint(ctxBuf[:], contextID)

	var lenBuf [10]byte
	l := binary.PutUvarint(lenBuf[:], uint64(len(payload)))

	if _, err := w.Write(ctxBuf[:n]); err != nil {
		globalMetrics.IncDropped()
		return err
	}

	if _, err := w.Write(lenBuf[:l]); err != nil {
		globalMetrics.IncDropped()
		return err
	}

	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			globalMetrics.IncDropped()
			return err
		}
	}

	globalMetrics.IncEncapsulated(len(payload))
	return nil
}

// ReadDatagram reads an RFC 9298 datagram frame from an io.Reader.
func ReadDatagram(r io.Reader, bufPool *BufferPool) (*Datagram, error) {
	contextID, err := readUvarint(r)
	if err != nil {
		return nil, err
	}

	length, err := readUvarint(r)
	if err != nil {
		return nil, err
	}

	var payload []byte
	if length > 0 {
		if length > 65535 { // Max UDP datagram size safety limit
			globalMetrics.IncDropped()
			return nil, errors.New("datagram payload length exceeds maximum UDP datagram limit")
		}
		if bufPool != nil {
			p := bufPool.Get()
			if uint64(len(p)) >= length {
				payload = p[:length]
			} else {
				bufPool.Put(p)
				payload = make([]byte, length)
			}
		} else {
			payload = make([]byte, length)
		}

		if _, err := io.ReadFull(r, payload); err != nil {
			globalMetrics.IncDropped()
			return nil, err
		}
	}

	globalMetrics.IncDecapsulated(len(payload))
	return &Datagram{
		ContextID: contextID,
		Payload:   payload,
	}, nil
}

func readUvarint(r io.Reader) (uint64, error) {
	var b [1]byte
	var x uint64
	var s uint
	for i := 0; i < 10; i++ {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		bVal := b[0]
		if bVal < 0x80 {
			if i == 9 && bVal > 1 {
				return 0, errors.New("overflow in uvarint")
			}
			return x | uint64(bVal)<<s, nil
		}
		x |= uint64(bVal&0x7f) << s
		s += 7
	}
	return 0, errors.New("overflow in uvarint")
}
