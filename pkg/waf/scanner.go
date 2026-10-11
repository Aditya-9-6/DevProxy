package waf

import (
	"io"
)

type Scanner struct {
	auto *Automaton
}

func NewScanner(patterns []string) (*Scanner, error) {
	bytePatterns := make([][]byte, len(patterns))
	for i, p := range patterns {
		bytePatterns[i] = []byte(p)
	}
	return &Scanner{auto: NewAutomaton(bytePatterns)}, nil
}

func (s *Scanner) ScanPayload(r io.Reader, callback func(string)) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			s.auto.Scan(buf[:n], func(idx int, pos int) {
				callback(string(s.auto.patterns[idx]))
			})
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
