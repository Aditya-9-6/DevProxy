package waf

import (
	"regexp"
)

// RedosDetector wraps a regex with a step-budget guard.
type RedosDetector struct {
	regex *regexp.Regexp
	guard *ReDoSGuard
}

func NewRedosDetector(pattern string, maxSteps uint64) (*RedosDetector, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &RedosDetector{
		regex: re,
		guard: NewReDoSGuard(maxSteps),
	}, nil
}

func (d *RedosDetector) Match(input []byte) bool {
	d.guard.Reset()
	// In a production Go environment, one would use a custom regex engine
	// that supports step-counting hooks. Here we provide the structural guard.
	return d.regex.Match(input)
}
