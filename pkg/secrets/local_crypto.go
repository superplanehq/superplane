package secrets

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
)

//
// Local secret payloads are bound to the secret ID, which never changes, so a
// rename is a metadata-only update. Older rows are bound to the name the secret
// had when the row was written. Those rows are rebound to the ID the next time
// the payload is written.
//

func associatedData(secretID uuid.UUID) []byte {
	return []byte(secretID.String())
}

func EncryptLocalData(ctx context.Context, encryptor crypto.Encryptor, secretID uuid.UUID, values map[string]string) ([]byte, error) {
	raw, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}

	return encryptor.Encrypt(ctx, raw, associatedData(secretID))
}

func DecryptLocalPayload(ctx context.Context, encryptor crypto.Encryptor, secret *models.Secret) ([]byte, error) {
	payload, err := encryptor.Decrypt(ctx, secret.Data, associatedData(secret.ID))
	if err == nil {
		return payload, nil
	}

	payload, legacyErr := encryptor.Decrypt(ctx, secret.Data, []byte(secret.Name))
	if legacyErr != nil {
		return nil, err
	}

	return payload, nil
}

func DecryptLocalData(ctx context.Context, encryptor crypto.Encryptor, secret *models.Secret) (map[string]string, error) {
	payload, err := DecryptLocalPayload(ctx, encryptor, secret)
	if err != nil {
		return nil, err
	}

	if len(payload) == 0 {
		return map[string]string{}, nil
	}

	var values map[string]string
	if err := json.Unmarshal(payload, &values); err != nil {
		return nil, err
	}

	return values, nil
}

// RebindLocalData returns the payload re-encrypted under the secret ID, or nil when
// the payload is already bound to it. Callers that rename a secret need this because
// the name a legacy payload is bound to is gone once the rename lands.
func RebindLocalData(ctx context.Context, encryptor crypto.Encryptor, secret *models.Secret) ([]byte, error) {
	if len(secret.Data) == 0 {
		return nil, nil
	}

	if _, err := encryptor.Decrypt(ctx, secret.Data, associatedData(secret.ID)); err == nil {
		return nil, nil
	}

	payload, err := encryptor.Decrypt(ctx, secret.Data, []byte(secret.Name))
	if err != nil {
		return nil, err
	}

	return encryptor.Encrypt(ctx, payload, associatedData(secret.ID))
}
