package multipart

import (
	"strings"
	"testing"
)

func TestBoundaryMatcher(t *testing.T) {
	bm := NewBoundaryMatcher("abc123xyz")
	if !bm.MatchFull([]byte("--abc123xyz\r\n")) {
		t.Errorf("Expected match full")
	}
	if !bm.MatchClose([]byte("--abc123xyz--\r\n")) {
		t.Errorf("Expected match close")
	}
}

func TestParserParseValid(t *testing.T) {
	parser := NewParser("boundary123", DefaultLimits())
	payload := "--boundary123\r\nContent-Type: text/plain\r\n\r\nsome body content\r\n--boundary123--\r\n"
	err := parser.Parse(strings.NewReader(payload))
	if err != nil {
		t.Errorf("Unexpected parse error: %v", err)
	}
}

func TestParserParseHeaderTooLarge(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxHeaderBytes = 10
	parser := NewParser("boundary123", limits)
	payload := "--boundary123\r\nContent-Type: text/plain\r\n\r\nbody\r\n--boundary123--\r\n"
	err := parser.Parse(strings.NewReader(payload))
	if err != ErrHeaderTooLarge {
		t.Errorf("Expected ErrHeaderTooLarge, got %v", err)
	}
}
