package secrets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
)

func EncryptLocalData(ctx context.Context, encryptor crypto.Encryptor, secret models.Secret, data map[string]string) ([]byte, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return encryptor.Encrypt(ctx, raw, secretAssociatedData(secret))
}

func DecryptLocalData(ctx context.Context, encryptor crypto.Encryptor, secret models.Secret) (map[string]string, error) {
	values, _, err := DecryptLocalDataWithFallback(ctx, encryptor, secret)
	return values, err
}

func DecryptLocalDataWithFallback(ctx context.Context, encryptor crypto.Encryptor, secret models.Secret) (map[string]string, bool, error) {
	plain, usedLegacyFallback, err := DecryptLocalDataRawWithFallback(ctx, encryptor, secret)
	if err != nil {
		return nil, false, err
	}

	if len(plain) == 0 {
		return make(map[string]string), usedLegacyFallback, nil
	}

	var values map[string]string
	if err := json.Unmarshal(plain, &values); err != nil {
		return nil, false, err
	}

	return values, usedLegacyFallback, nil
}

func DecryptLocalDataRaw(ctx context.Context, encryptor crypto.Encryptor, secret models.Secret) ([]byte, error) {
	plain, _, err := DecryptLocalDataRawWithFallback(ctx, encryptor, secret)
	return plain, err
}

func DecryptLocalDataRawWithFallback(ctx context.Context, encryptor crypto.Encryptor, secret models.Secret) ([]byte, bool, error) {
	plain, primaryErr := encryptor.Decrypt(ctx, secret.Data, secretAssociatedData(secret))
	if primaryErr == nil {
		return plain, false, nil
	}

	if !shouldTryLegacyNameAssociatedData(secret) {
		return nil, false, primaryErr
	}

	plain, legacyErr := encryptor.Decrypt(ctx, secret.Data, []byte(secret.Name))
	if legacyErr == nil {
		return plain, true, nil
	}

	return nil, false, fmt.Errorf("decrypt with secret ID associated data failed: %w; decrypt with legacy name associated data failed: %v", primaryErr, legacyErr)
}

func secretAssociatedData(secret models.Secret) []byte {
	if secret.ID != uuid.Nil {
		return []byte(secret.ID.String())
	}

	return []byte(secret.Name)
}

func shouldTryLegacyNameAssociatedData(secret models.Secret) bool {
	return secret.ID != uuid.Nil && secret.Name != ""
}
