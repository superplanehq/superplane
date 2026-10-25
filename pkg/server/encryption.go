package server

import (
	"context"
	"fmt"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryption"
)

type applicationEncryption struct {
	encryptor crypto.Encryptor
	keyStore  *encryption.Store
}

func loadApplicationEncryption(ctx context.Context) (applicationEncryption, error) {
	masters, primaryRef, err := crypto.MasterKeysFromEnv(ctx)
	if err != nil {
		return applicationEncryption{}, fmt.Errorf("load encryption master keys: %w", err)
	}
	if masters == nil {
		return applicationEncryption{}, fmt.Errorf("ENCRYPTION_MASTER_KEYS is required")
	}

	keyStore := encryption.NewStore(masters, primaryRef)
	if err := keyStore.Initialize(ctx); err != nil {
		return applicationEncryption{}, fmt.Errorf("load application data keys: %w", err)
	}
	versioned := crypto.NewVersionedEncryptor(keyStore)
	return applicationEncryption{encryptor: encryption.NewApplicationEncryptor(versioned, keyStore), keyStore: keyStore}, nil
}
