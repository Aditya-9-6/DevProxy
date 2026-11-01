package http2priority

// CreditManager handles stream resource allocation and prioritization credits.
type CreditManager struct {
	defaultCredits int
}

// NewCreditManager creates a new CreditManager.
func NewCreditManager(defaultCredits int) *CreditManager {
	return &CreditManager{
		defaultCredits: defaultCredits,
	}
}

// GetCredits returns initial credits for a stream.
func (cm *CreditManager) GetCredits() int {
	return cm.defaultCredits
}
