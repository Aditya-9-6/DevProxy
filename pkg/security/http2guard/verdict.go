package http2guard

// Verdict represents the admission decision for an HTTP/2 frame or stream action.
type Verdict uint8

const (
	VerdictAllow Verdict = iota
	VerdictRateLimited
	VerdictBlocked
)

func (v Verdict) String() string {
	switch v {
	case VerdictAllow:
		return "ALLOW"
	case VerdictRateLimited:
		return "RATE_LIMITED"
	case VerdictBlocked:
		return "BLOCKED"
	default:
		return "UNKNOWN"
	}
}
