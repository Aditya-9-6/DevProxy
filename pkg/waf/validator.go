package waf

import (
	"sync"
)

var pool = sync.Pool{
	New: func() interface{} { return &BitVector{} },
}

// ValidateSQLi performs a zero-allocation semantic check on input strings using a state machine.
func ValidateSQLi(input string) bool {
	vec := pool.Get().(*BitVector)
	defer func() {
		vec.Reset()
		pool.Put(vec)
	}()

	state := 0
	for i := 0; i < len(input); i++ {
		c := input[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}

		// Check for comments or url-encoded comment indicators (-- or # or %2d%2d)
		if (c == '-' && i+1 < len(input) && input[i+1] == '-') || c == '#' || (c == '%' && i+2 < len(input) && (input[i+1]|32) == '2' && (input[i+2]|32) == 'd') {
			vec.Set(uint(TokenComment))
			return true
		}

		// Check for basic tautology patterns like 1=1 or similar numeric equivalents
		if c == '1' && i+2 < len(input) && input[i+1] == '=' && input[i+2] == '1' {
			vec.Set(uint(TokenTautology))
			return true
		}

		switch state {
		case 0:
			if c == 'o' || c == 's' || c == 'u' || c == '1' || c == 'r' || c == 'a' {
				state = 1
			}
		case 1:
			if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '=' {
				continue
			}
			if c == '1' && i+2 < len(input) && input[i+1] == '=' && input[i+2] == '1' {
				vec.Set(uint(TokenTautology))
				return true
			}
			state = 0
		}
	}
	return vec.Get(uint(TokenTautology)) || vec.Get(uint(TokenComment))
}
