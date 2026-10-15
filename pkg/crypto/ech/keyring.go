package ech

import "sync/atomic"

type KeyRing struct {
	activeKey atomic.Pointer[[]byte]
}

func (k *KeyRing) Rotate(newKey []byte) {
	k.activeKey.Store(&newKey)
}

func (k *KeyRing) Get() []byte {
	ptr := k.activeKey.Load()
	if ptr == nil {
		return nil
	}
	return *ptr
}
