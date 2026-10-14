package waf

// IsPredictable checks if a cookie value has suspiciously low entropy.
func IsPredictable(entropy float64) bool {
	// Threshold for session token predictability
	return entropy < 2.5
}
