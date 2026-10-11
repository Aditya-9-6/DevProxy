package waf

import (
	"io"
)

type Scanner struct {
	auto *Automaton
}

func NewScanner(patterns []string) (*Scanner, error) {
	auto, err := NewAutomaton(patterns)
	if err != nil {
		return nil, err
	}
	return &Scanner{auto: auto}, nil
}

func (s *Scanner) ScanPayload(r io.Reader, callback func(string)) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	for {
		n, err := r.Read(buf[:cap(buf)])
		if n > 0 {
			s.auto.Scan(buf[:n], callback)
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}
