package tls13

import (
	"crypto"
	"testing"
)

func TestKeySchedule_Derivations(t *testing.T) {
	ks := NewKeySchedule(crypto.SHA256)
	ks.DeriveEarlySecret(nil)

	ecdhe := make([]byte, 32)
	transcriptSum := make([]byte, 32)
	ks.DeriveHandshakeSecrets(ecdhe, transcriptSum)

	if len(ks.ClientHandshakeTrafficSecret()) != 32 {
		t.Errorf("invalid client handshake traffic secret length")
	}
	if len(ks.ServerHandshakeTrafficSecret()) != 32 {
		t.Errorf("invalid server handshake traffic secret length")
	}

	ks.DeriveMasterSecret(transcriptSum)
	if len(ks.ClientAppTrafficSecret()) != 32 {
		t.Errorf("invalid client app traffic secret length")
	}
	if len(ks.ServerAppTrafficSecret()) != 32 {
		t.Errorf("invalid server app traffic secret length")
	}
	if len(ks.ResumptionMasterSecret()) != 32 {
		t.Errorf("invalid resumption master secret length")
	}

	exporter := ks.ExporterSecret("test label", nil, 32)
	if len(exporter) != 32 {
		t.Errorf("invalid exporter secret length")
	}
}
