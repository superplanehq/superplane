package contexts

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	secretstore "github.com/superplanehq/superplane/pkg/secrets"
	"gorm.io/gorm"
)

// SecretsContext resolves organization secret key values for component execution.
type SecretsContext struct {
	tx             *gorm.DB
	organizationID uuid.UUID
	encryptor      crypto.Encryptor
}

// NewSecretsContext returns a SecretsContext that looks up secrets in the given transaction
// for the given organization.
func NewSecretsContext(tx *gorm.DB, organizationID uuid.UUID, encryptor crypto.Encryptor) *SecretsContext {
	return &SecretsContext{
		tx:             tx,
		organizationID: organizationID,
		encryptor:      encryptor,
	}
}

// GetKey implements core.SecretsContext.
func (c *SecretsContext) GetKey(secretName, keyName string) ([]byte, error) {
	if secretName == "" || keyName == "" {
		return nil, core.ErrSecretKeyNotFound
	}

	secret, err := models.FindSecretByNameInTransaction(c.tx, models.DomainTypeOrganization, c.organizationID, secretName)
	if err != nil {
		return nil, err
	}

	data, err := c.decryptSecretData(secret)
	if err != nil {
		return nil, err
	}

	val, ok := data[keyName]
	if !ok || val == "" {
		return nil, core.ErrSecretKeyNotFound
	}

	return []byte(val), nil
}

func (c *SecretsContext) decryptSecretData(secret *models.Secret) (map[string]string, error) {
	return secretstore.DecryptLocalData(context.Background(), c.encryptor, *secret)
}
