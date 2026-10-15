package proxy

// Header bitmask constants for O(1) state machine transitions.
const (
	MaskContentLength    uint64 = 1 << 0
	MaskTransferEncoding uint64 = 1 << 1
	MaskConnection       uint64 = 1 << 2
	MaskUpgrade          uint64 = 1 << 3
)

// DependencyTable defines invalid bitmask combinations (e.g., Content-Length + Transfer-Encoding).
var DependencyTable = []struct {
	Mask uint64
	Err  string
}{
	{MaskContentLength | MaskTransferEncoding, "Transfer-Encoding and Content-Length cannot coexist"},
}
