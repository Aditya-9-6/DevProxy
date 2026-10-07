package analysis

import (
	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
)

type SecurityRulesEngine struct {
	rules []Rule
}

type Rule interface {
	Evaluate(event *ringbuffer.TrafficEvent) []*Finding
}

func NewSecurityRulesEngine() *SecurityRulesEngine {
	return &SecurityRulesEngine{rules: make([]Rule, 0)}
}

func (e *SecurityRulesEngine) AddRules(rules []Rule) {
	e.rules = append(e.rules, rules...)
}

func (e *SecurityRulesEngine) AddRule(rule Rule) {
	e.rules = append(e.rules, rule)
}

func (e *SecurityRulesEngine) Analyze(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding
	for _, rule := range e.rules {
		findings = append(findings, rule.Evaluate(event)...)
	}
	return findings
}

func LoadCustomRulesFromFile(path string) ([]Rule, error) {
	// Implementation stub for loading rules from file
	return nil, nil
}
