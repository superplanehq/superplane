package secrets

import (
	"context"
	"encoding/json"

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
	plain, err := DecryptLocalDataRaw(ctx, encryptor, secret)
	if err != nil {
		return nil, err
	}

	if len(plain) == 0 {
		return make(map[string]string), nil
	}

	var values map[string]string
	if err := json.Unmarshal(plain, &values); err != nil {
		return nil, err
	}

	return values, nil
}

func DecryptLocalDataRaw(ctx context.Context, encryptor crypto.Encryptor, secret models.Secret) ([]byte, error) {
	plain, err := encryptor.Decrypt(ctx, secret.Data, secretAssociatedData(secret))
	if err != nil && secret.ID != uuid.Nil && secret.Name != "" {
		plain, err = encryptor.Decrypt(ctx, secret.Data, []byte(secret.Name))
	}

	return plain, err
}

func secretAssociatedData(secret models.Secret) []byte {
	if secret.ID != uuid.Nil {
		return []byte(secret.ID.String())
	}

	return []byte(secret.Name)
}
