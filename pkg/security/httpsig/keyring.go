package httpsig

// Keyring manages keys for verification.
type Keyring struct {
	keys map[string][]byte
}

// NewKeyring creates an empty Keyring.
func NewKeyring() *Keyring {
	return &Keyring{keys: make(map[string][]byte)}
}

// AddKey adds a key to the keyring.
func (k *Keyring) AddKey(id string, pubKey []byte) {
	k.keys[id] = pubKey
}

// GetKey retrieves a public key.
func (k *Keyring) GetKey(id string) ([]byte, bool) {
	p, ok := k.keys[id]
	return p, ok
}
