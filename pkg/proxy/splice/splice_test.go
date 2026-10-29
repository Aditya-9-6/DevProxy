package splice

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestSpliceCopy(t *testing.T) {
	c1, s1 := net.Pipe()
	defer c1.Close()
	defer s1.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data := []byte("testing splice data transfer")

	go func() {
		_, _ = c1.Write(data)
		_ = c1.Close()
	}()

	var readConn net.Conn
	c2, s2 := net.Pipe()
	defer c2.Close()
	defer s2.Close()
	readConn = s2

	go func() {
		_, _ = SpliceCopy(ctx, c2, s1)
		_ = c2.Close()
	}()

	received := make([]byte, len(data))
	n, err := readConn.Read(received)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(received[:n]) != string(data) {
		t.Fatalf("expected '%s', got '%s'", string(data), string(received[:n]))
	}
}
