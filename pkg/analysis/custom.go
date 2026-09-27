package analysis

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/Aditya-9-6/DevProxy/pkg/ringbuffer"
	"gopkg.in/yaml.v3"
)

// CustomRuleConfig represents a single rule definition in YAML.
type CustomRuleConfig struct {
	Name        string `yaml:"name"`
	Category    string `yaml:"category"`
	Severity    string `yaml:"severity"`
	Regex       string `yaml:"regex"`
	Description string `yaml:"description"`
	Remediation string `yaml:"remediation"`
	Location    string `yaml:"location"` // "all", "request_body", "response_body", "headers", "url"
}

// CustomRulesDocument is the top-level YAML file format.
type CustomRulesDocument struct {
	Rules []CustomRuleConfig `yaml:"rules"`
}

// CustomRule evaluates traffic against user-defined regexes from YAML.
type CustomRule struct {
	cfg   CustomRuleConfig
	regex *regexp.Regexp
}

// NewCustomRule constructs a CustomRule from config.
func NewCustomRule(cfg CustomRuleConfig) (*CustomRule, error) {
	if cfg.Severity == "" {
		cfg.Severity = SeverityHigh
	}
	if cfg.Category == "" {
		cfg.Category = "CUSTOM_RULE"
	}
	if cfg.Location == "" {
		cfg.Location = "all"
	}

	re, err := regexp.Compile(cfg.Regex)
	if err != nil {
		return nil, fmt.Errorf("invalid regex for rule '%s': %w", cfg.Name, err)
	}

	return &CustomRule{
		cfg:   cfg,
		regex: re,
	}, nil
}

func (r *CustomRule) Name() string {
	return r.cfg.Name
}

func (r *CustomRule) Evaluate(event *ringbuffer.TrafficEvent) []*Finding {
	var findings []*Finding
	loc := strings.ToLower(r.cfg.Location)

	// Check URL
	if loc == "all" || loc == "url" {
		if match := r.regex.FindString(event.URL); match != "" {
			findings = append(findings, r.makeFinding(event, "Request URL", match))
		}
	}

	// Check Request Body
	if (loc == "all" || loc == "request_body") && len(event.ReqBody) > 0 {
		if match := r.regex.Find(event.ReqBody); match != nil {
			evidence := string(match)
			if len(evidence) > 60 {
				evidence = evidence[:30] + "..." + evidence[len(evidence)-15:]
			}
			findings = append(findings, r.makeFinding(event, "Request Body", evidence))
		}
	}

	// Check Response Body
	if (loc == "all" || loc == "response_body") && len(event.RespBody) > 0 {
		if match := r.regex.Find(event.RespBody); match != nil {
			evidence := string(match)
			if len(evidence) > 60 {
				evidence = evidence[:30] + "..." + evidence[len(evidence)-15:]
			}
			findings = append(findings, r.makeFinding(event, "Response Body", evidence))
		}
	}

	// Check Headers
	if loc == "all" || loc == "headers" {
		for k, vv := range event.ReqHeaders {
			for _, v := range vv {
				if r.regex.MatchString(v) {
					findings = append(findings, r.makeFinding(event, "Request Header: "+k, v))
				}
			}
		}
		for k, vv := range event.RespHeaders {
			for _, v := range vv {
				if r.regex.MatchString(v) {
					findings = append(findings, r.makeFinding(event, "Response Header: "+k, v))
				}
			}
		}
	}

	return findings
}

func (r *CustomRule) makeFinding(event *ringbuffer.TrafficEvent, location, evidence string) *Finding {
	return &Finding{
		RequestID:   event.ID,
		Timestamp:   event.Timestamp,
		Severity:    strings.ToUpper(r.cfg.Severity),
		Category:    r.cfg.Category,
		RuleName:    r.cfg.Name,
		Title:       fmt.Sprintf("%s matched in %s", r.cfg.Name, location),
		Description: r.cfg.Description,
		Evidence:    evidence,
		Location:    location,
		Remediation: r.cfg.Remediation,
		URL:         event.URL,
		Method:      event.Method,
	}
}

// LoadCustomRulesFromFile parses a YAML rules file and compiles each custom rule.
func LoadCustomRulesFromFile(path string) ([]Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadCustomRulesFromYAML(data)
}

// LoadCustomRulesFromYAML parses YAML bytes into a list of Rule instances.
func LoadCustomRulesFromYAML(data []byte) ([]Rule, error) {
	var doc CustomRulesDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom rules YAML: %w", err)
	}

	var rules []Rule
	for _, cfg := range doc.Rules {
		r, err := NewCustomRule(cfg)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, nil
}
