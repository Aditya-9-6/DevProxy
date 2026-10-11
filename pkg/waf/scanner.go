package waf

import "sync"

type Scanner struct {
	auto *Automaton
}

var (
	instance *Scanner
	once     sync.Once
)

func NewScanner() *Scanner {
	once.Do(func() {
		patterns := []string{"SELECT", "UNION", "<script>", "alert(", "DROP TABLE"}
		instance = &Scanner{auto: NewAutomaton(patterns)}
	})
	return instance
}

func (s *Scanner) ScanPayload(data []byte) []string {
	var found []string
	s.auto.Scan(data, func(match string) {
		found = append(found, match)
	})
	return found
}
