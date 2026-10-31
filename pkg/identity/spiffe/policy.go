package spiffe

type Policy struct {
	AllowedTrustDomains []string
	AllowedPaths        []string
}

func (p Policy) Evaluate(id ID) bool {
	if len(p.AllowedTrustDomains) > 0 {
		allowed := false
		for _, td := range p.AllowedTrustDomains {
			if id.TrustDomain == td {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	if len(p.AllowedPaths) > 0 {
		allowed := false
		for _, path := range p.AllowedPaths {
			if id.Path == path {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}
	return true
}
