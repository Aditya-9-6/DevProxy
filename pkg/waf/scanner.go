package waf

type Scanner struct {
	auto *Automaton
}

func NewScanner(patterns []string) *Scanner {
	return &Scanner{auto: NewAutomaton(patterns)}
}

func (s *Scanner) ScanPayload(data []byte, callback func(string)) {
	s.auto.Scan(data, callback)
}
