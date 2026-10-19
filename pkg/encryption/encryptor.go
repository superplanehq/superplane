package encryption

import (
	"context"

	"github.com/superplanehq/superplane/pkg/crypto"
	"gorm.io/gorm"
)

type applicationEncryptor struct {
	versioned *crypto.VersionedEncryptor
	store     *Store
}

func NewApplicationEncryptor(versioned *crypto.VersionedEncryptor, store *Store) crypto.Encryptor {
	return &applicationEncryptor{versioned: versioned, store: store}
}

func (e *applicationEncryptor) Encrypt(ctx context.Context, plain, associatedData []byte) ([]byte, error) {
	return e.versioned.Encrypt(ctx, plain, associatedData)
}

func (e *applicationEncryptor) Decrypt(ctx context.Context, value, associatedData []byte) ([]byte, error) {
	return e.versioned.Decrypt(ctx, value, associatedData)
}

func (e *applicationEncryptor) ForDB(tx *gorm.DB) crypto.Encryptor {
	return &applicationEncryptor{
		versioned: e.versioned.WithKeys(transactionDataKeys{store: e.store, tx: tx}),
		store:     e.store,
	}
}

// ForDB binds application encryption key lookups to the caller's DB handle.
// Legacy and no-op encryptors do not query the data key table.
func ForDB(tx *gorm.DB, encryptor crypto.Encryptor) crypto.Encryptor {
	if tx == nil {
		return encryptor
	}
	if scoped, ok := encryptor.(interface {
		ForDB(*gorm.DB) crypto.Encryptor
	}); ok {
		return scoped.ForDB(tx)
	}
	return encryptor
}
