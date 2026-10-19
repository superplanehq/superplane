package server

import (
	"context"
	"fmt"
	"os"

	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryption"
)

type applicationEncryption struct {
	encryptor    crypto.Encryptor
	keyStore     *encryption.Store
	legacyWrites bool
}

func loadApplicationEncryption(ctx context.Context) (applicationEncryption, error) {
	legacy, err := crypto.LegacyDecryptorFromEnv()
	if err != nil {
		return applicationEncryption{}, fmt.Errorf("load legacy encryption: %w", err)
	}
	masters, primaryRef, err := crypto.MasterKeysFromEnv(ctx)
	if err != nil {
		return applicationEncryption{}, fmt.Errorf("load encryption master keys: %w", err)
	}
	writeFormat := os.Getenv("ENCRYPTION_WRITE_FORMAT")
	if writeFormat != "" && writeFormat != "legacy" && writeFormat != "versioned" {
		return applicationEncryption{}, fmt.Errorf("ENCRYPTION_WRITE_FORMAT must be legacy or versioned")
	}
	if writeFormat == "legacy" && (masters == nil || legacy == nil || os.Getenv("ENCRYPTION_LEGACY_FORMAT") == "plaintext") {
		return applicationEncryption{}, fmt.Errorf("legacy writes require ENCRYPTION_MASTER_KEYS and an AES-GCM ENCRYPTION_KEY")
	}
	if writeFormat == "versioned" && masters == nil {
		return applicationEncryption{}, fmt.Errorf("versioned writes require ENCRYPTION_MASTER_KEYS")
	}
	if masters == nil {
		if legacy == nil {
			return applicationEncryption{}, fmt.Errorf("ENCRYPTION_KEY or ENCRYPTION_MASTER_KEYS is required")
		}
		if os.Getenv("ENCRYPTION_LEGACY_FORMAT") == "plaintext" {
			return applicationEncryption{}, fmt.Errorf("ENCRYPTION_MASTER_KEYS is required to convert plaintext values")
		}
		return applicationEncryption{encryptor: legacy}, nil
	}

	keyStore := encryption.NewStore(masters, primaryRef)
	if err := keyStore.Initialize(ctx); err != nil {
		return applicationEncryption{}, fmt.Errorf("load application data keys: %w", err)
	}
	versioned := crypto.NewVersionedEncryptor(keyStore, legacy)
	if writeFormat == "legacy" {
		return applicationEncryption{encryptor: encryption.NewApplicationEncryptor(versioned.WithLegacyWrites(), keyStore), keyStore: keyStore, legacyWrites: true}, nil
	}
	return applicationEncryption{encryptor: encryption.NewApplicationEncryptor(versioned, keyStore), keyStore: keyStore}, nil
}
