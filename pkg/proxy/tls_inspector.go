package proxy

import (
	"crypto/tls"
	"github.com/Aditya-9-6/DevProxy/pkg/security"
)

// TLSInspector handles auditing of TLS handshakes.
type TLSInspector struct {
	Auditor *security.CipherAuditor
}

func NewTLSInspector() *TLSInspector {
	return &TLSInspector{
		Auditor: security.NewCipherAuditor(),
	}
}

// InspectHandshake records the cipher suite used in the connection.
func (i *TLSInspector) InspectHandshake(state tls.ConnectionState) {
	if i.Auditor != nil {
		i.Auditor.Record(state.CipherSuite)
	}
}
