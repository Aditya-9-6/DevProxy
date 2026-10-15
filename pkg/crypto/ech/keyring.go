package ech

import "sync/atomic"

type KeyRing struct {
	activeKey atomic.Pointer[[]byte]
}

func (k *KeyRing) Rotate(newKey []byte) {
	if newKey == nil {
		k.activeKey.Store(nil)
		return
	}
	keyCopy := make([]byte, len(newKey))
	copy(keyCopy, newKey)
	k.activeKey.Store(&keyCopy)
}

func (k *KeyRing) Get() []byte {
	ptr := k.activeKey.Load()
	if ptr == nil {
		return nil
	}
	return *ptr
}
