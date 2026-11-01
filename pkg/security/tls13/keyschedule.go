package tls13

import (
	"crypto"
)

// KeySchedule manages the TLS 1.3 key derivation hierarchy (RFC 8446 Section 7.1).
type KeySchedule struct {
	hashFunc          crypto.Hash
	zero              []byte
	earlySecret       []byte
	boundSecret       []byte
	handshakeSecret   []byte
	clientHandshakeTrafficSecret []byte
	serverHandshakeTrafficSecret []byte
	masterSecret      []byte
	clientAppTrafficSecret       []byte
	serverAppTrafficSecret       []byte
	resumptionMasterSecret       []byte
}

// NewKeySchedule initializes a KeySchedule with the given hash function.
func NewKeySchedule(hashFunc crypto.Hash) *KeySchedule {
	return &KeySchedule{
	hashFunc: hashFunc,
	zero:     make([]byte, hashFunc.Size()),
}
}

// DeriveEarlySecret derives the early secret given an optional psk.
func (ks *KeySchedule) DeriveEarlySecret(psk []byte) {
	if psk == nil {
		psk = ks.zero
	}
	ks.earlySecret = Extract(ks.hashFunc, nil, psk)
}

// DeriveHandshakeSecrets derives the handshake secret from the early secret and ECDHE shared secret.
func (ks *KeySchedule) DeriveHandshakeSecrets(ecdhe []byte, transcriptHash []byte) {
	if ks.earlySecret == nil {
		ks.earlySecret = ks.zero
	}
	derived := ExpandLabel(ks.hashFunc, ks.earlySecret, "derived", nil, ks.hashFunc.Size())
	ks.handshakeSecret = Extract(ks.hashFunc, derived, ecdhe)

	ks.clientHandshakeTrafficSecret = ExpandLabel(ks.hashFunc, ks.handshakeSecret, "c hs ts", transcriptHash, ks.hashFunc.Size())
	ks.serverHandshakeTrafficSecret = ExpandLabel(ks.hashFunc, ks.handshakeSecret, "s hs ts", transcriptHash, ks.hashFunc.Size())
}

// DeriveMasterSecret derives the master secret and application traffic secrets.
func (ks *KeySchedule) DeriveMasterSecret(transcriptHash []byte) {
	if ks.handshakeSecret == nil {
		ks.handshakeSecret = ks.zero
	}
	derived := ExpandLabel(ks.hashFunc, ks.handshakeSecret, "derived", nil, ks.hashFunc.Size())
	ks.masterSecret = Extract(ks.hashFunc, derived, ks.zero)

	ks.clientAppTrafficSecret = ExpandLabel(ks.hashFunc, ks.masterSecret, "c ap ts", transcriptHash, ks.hashFunc.Size())
	ks.serverAppTrafficSecret = ExpandLabel(ks.hashFunc, ks.masterSecret, "s ap ts", transcriptHash, ks.hashFunc.Size())

	ks.resumptionMasterSecret = ExpandLabel(ks.hashFunc, ks.masterSecret, "res master", transcriptHash, ks.hashFunc.Size())
}

// ExporterSecret derives exporter key material using HKDF-Expand-Label.
func (ks *KeySchedule) ExporterSecret(label string, context []byte, length int) []byte {
	return ExpandLabel(ks.hashFunc, ks.masterSecret, label, context, length)
}

// Accessor methods for traffic secrets
func (ks *KeySchedule) ClientHandshakeTrafficSecret() []byte { return ks.clientHandshakeTrafficSecret }
func (ks *KeySchedule) ServerHandshakeTrafficSecret() []byte { return ks.serverHandshakeTrafficSecret }
func (ks *KeySchedule) ClientAppTrafficSecret() []byte       { return ks.clientAppTrafficSecret }
func (ks *KeySchedule) ServerAppTrafficSecret() []byte       { return ks.serverAppTrafficSecret }
func (ks *KeySchedule) ResumptionMasterSecret() []byte       { return ks.resumptionMasterSecret }
