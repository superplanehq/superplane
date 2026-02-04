package contexts

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// NewSecretResolver returns a SecretResolver that looks up secrets by ID in the given
// transaction and domain, decrypts the secret data, and returns it as a map.
// The resolved value is the secret's local data (map of key-value pairs).
func NewSecretResolver(tx *gorm.DB, domainType string, domainID uuid.UUID, encryptor crypto.Encryptor) SecretResolver {
	return func(secretID string) (any, error) {
		secret, err := models.FindSecretByIDInTransaction(tx, domainType, domainID, secretID)
		if err != nil {
			return nil, err
		}

		decrypted, err := encryptor.Decrypt(context.Background(), secret.Data, []byte(secret.Name))
		if err != nil {
			return nil, err
		}

		var data map[string]string
		if err := json.Unmarshal(decrypted, &data); err != nil {
			return nil, err
		}

		// Return as map[string]any for consistency with JSON config
		result := make(map[string]any, len(data))
		for k, v := range data {
			result[k] = v
		}
		return result, nil
	}
}
