package proxy

import (
	"net/http"
	"strings"
)

// IsGRPC checks if the content type indicates gRPC traffic.
func IsGRPC(header http.Header) bool {
	ct := header.Get("Content-Type")
	return strings.HasPrefix(ct, "application/grpc")
}
