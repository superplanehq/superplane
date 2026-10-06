package licensing

import (
	"crypto/ecdsa"
	"errors"
	"sync/atomic"
)

// ErrKeyListDowngrade is returned for a key list that is older than the
// trusted one. Accepting it could restore a key that the issuer removed.
var ErrKeyListDowngrade = errors.New("license key list is older than the trusted list")

// KeyStore holds the license signing keys that SuperPlane trusts now. Only a
// key list signed by a root key can change them, and the version only grows.
type KeyStore struct {
	roots   *KeySet
	extra   *KeySet
	current atomic.Pointer[KeyList]
}

// NewKeyStore trusts the bootstrap key list when it is not empty. The extra
// keys are trusted in addition to every key list; only development builds
// pass them.
func NewKeyStore(roots *KeySet, bootstrap []byte, extra *KeySet) (*KeyStore, error) {
	store := &KeyStore{roots: roots, extra: extra}
	if len(bootstrap) == 0 {
		return store, nil
	}

	list, err := VerifyKeyList(bootstrap, roots)
	if err != nil {
		return nil, err
	}

	store.current.Store(list)
	return store, nil
}

func (s *KeyStore) Roots() *KeySet {
	return s.roots
}

// Current returns the trusted key list, or nil when there is none.
func (s *KeyStore) Current() *KeyList {
	return s.current.Load()
}

// Update trusts a verified key list when it is newer than the current one. It
// reports whether the trusted keys changed.
func (s *KeyStore) Update(list *KeyList) (bool, error) {
	for {
		current := s.current.Load()
		if current != nil && list.Version < current.Version {
			return false, ErrKeyListDowngrade
		}

		if current != nil && list.Version == current.Version {
			return false, nil
		}

		if s.current.CompareAndSwap(current, list) {
			return true, nil
		}
	}
}

func (s *KeyStore) PublicKey(keyID string) (*ecdsa.PublicKey, bool) {
	if s.extra != nil {
		if publicKey, ok := s.extra.PublicKey(keyID); ok {
			return publicKey, true
		}
	}

	current := s.current.Load()
	if current == nil {
		return nil, false
	}

	return current.Keys.PublicKey(keyID)
}
