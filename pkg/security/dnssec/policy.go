package dnssec

// Policy defines DNSSEC enforcement policy.
type Policy int

const (
	PolicyStrict Policy = iota
	PolicyPermissive
	PolicyDisabled
)

// Config holds DNSSEC security configurations.
type Config struct {
	Policy Policy
}

// DefaultConfig returns a secure default DNSSEC configuration.
func DefaultConfig() *Config {
	return &Config{
		Policy: PolicyPermissive,
	}
}
