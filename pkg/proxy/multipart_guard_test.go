package proxy

import (
	"strings"
	"testing"
	"github.com/Aditya-9-6/DevProxy/pkg/security/multipart"
)

func TestMultipartGuard(t *testing.T) {
	guard := NewMultipartGuard("boundary123", multipart.DefaultLimits())
	err := guard.Validate(strings.NewReader("--boundary123\r\nContent-Type: text/plain\r\n\r\nhello\r\n--boundary123--\r\n"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
