package tunnel

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func TestSpliceCopy(t *testing.T) {
	// Setup a local TCP listener to act as the "destination" server
	dstListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on destination: %v", err)
	}
	defer dstListener.Close()

	var wg sync.WaitGroup
	wg.Add(1)

	var dstServerConn *net.TCPConn
	go func() {
		defer wg.Done()
		conn, err := dstListener.Accept()
		if err != nil {
			t.Errorf("failed to accept on destination: %v", err)
			return
		}
		dstServerConn = conn.(*net.TCPConn)
	}()

	// Setup a local TCP listener to act as the "source" server
	srcListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on source: %v", err)
	}
	defer srcListener.Close()

	wg.Add(1)
	var srcServerConn *net.TCPConn
	go func() {
		defer wg.Done()
		conn, err := srcListener.Accept()
		if err != nil {
			t.Errorf("failed to accept on source: %v", err)
			return
		}
		srcServerConn = conn.(*net.TCPConn)
	}()

	// Connect clients to both servers
	dstClientConn, err := net.Dial("tcp", dstListener.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial destination: %v", err)
	}
	defer dstClientConn.Close()

	srcClientConn, err := net.Dial("tcp", srcListener.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial source: %v", err)
	}
	defer srcClientConn.Close()

	// Wait for server connections to be accepted
	wg.Wait()
	if dstServerConn == nil || srcServerConn == nil {
		t.Fatal("server connections not established")
	}
	defer dstServerConn.Close()
	defer srcServerConn.Close()

	// Prepare data to send
	testData := []byte("Hello, SpliceCopy!")

	// Start SpliceCopy in a goroutine
	// We want to copy FROM srcClientConn TO dstClientConn
	errCh := make(chan error, 1)
	writtenCh := make(chan int64, 1)
	go func() {
		written, err := SpliceCopy(dstClientConn.(*net.TCPConn), srcClientConn.(*net.TCPConn))
		writtenCh <- written
		errCh <- err
	}()

	// Write data to the source server connection. This data will be read by srcClientConn,
	// spliced to dstClientConn, and finally read by dstServerConn.
	go func() {
		_, err := srcServerConn.Write(testData)
		if err != nil {
			t.Errorf("failed to write to source server: %v", err)
		}
		// Close the write side to signal EOF to the splice
		srcServerConn.CloseWrite()
	}()

	// Read data from the destination server connection.
	var buf bytes.Buffer
	readErrCh := make(chan error, 1)
	go func() {
		// Just read what we expect
		io.CopyN(&buf, dstServerConn, int64(len(testData)))
		readErrCh <- nil
	}()

	// Wait for SpliceCopy to finish
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("SpliceCopy failed: %v", err)
		}
		written := <-writtenCh
		if written != int64(len(testData)) {
			t.Errorf("expected %d bytes written, got %d", len(testData), written)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SpliceCopy timed out")
	}

	// Wait for the read to finish
	select {
	case err := <-readErrCh:
		if err != nil && err != io.EOF {
			t.Errorf("read from destination server failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("read timed out")
	}

	if !bytes.Equal(buf.Bytes(), testData) {
		t.Errorf("expected data %q, got %q", testData, buf.Bytes())
	}
}

func TestSpliceCopyError(t *testing.T) {
	// Try calling SpliceCopy with a closed connection to test error handling
	srcListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on source: %v", err)
	}
	defer srcListener.Close()

	srcClientConn, err := net.Dial("tcp", srcListener.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial source: %v", err)
	}

	// Close to induce an error
	srcClientConn.Close()

	dstListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on destination: %v", err)
	}
	defer dstListener.Close()

	dstClientConn, err := net.Dial("tcp", dstListener.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial destination: %v", err)
	}
	defer dstClientConn.Close()

	_, err = SpliceCopy(dstClientConn.(*net.TCPConn), srcClientConn.(*net.TCPConn))
	if err == nil {
		t.Error("expected error from SpliceCopy with closed source connection, got nil")
	}
}
