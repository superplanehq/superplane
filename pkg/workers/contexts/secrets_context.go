package contexts

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

/*
 * SecretsContext implements core.SecretsContext,
 * resolving organization secret key values for component execution.
 */
type SecretsContext struct {
	tx             *gorm.DB
	registry       *registry.Registry
	encryptor      crypto.Encryptor
	organizationID uuid.UUID
}

func NewSecretsContext(tx *gorm.DB, reg *registry.Registry, organizationID uuid.UUID, encryptor crypto.Encryptor) *SecretsContext {
	return &SecretsContext{
		tx:             tx,
		encryptor:      encryptor,
		registry:       reg,
		organizationID: organizationID,
	}
}

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

func (c *SecretsContext) GetSecretKeys(secretName string) (map[string][]byte, error) {
	secretName = strings.TrimSpace(secretName)
	if secretName == "" {
		return nil, fmt.Errorf("secret name is required")
	}

	secret, err := models.FindSecretByNameInTransaction(c.tx, models.DomainTypeOrganization, c.organizationID, secretName)
	if err != nil {
		return nil, err
	}

	data, err := c.decryptSecretData(secret)
	if err != nil {
		return nil, err
	}

	keys := make(map[string][]byte, len(data))
	for name, value := range data {
		name = strings.TrimSpace(name)
		if name == "" || value == "" {
			continue
		}
		keys[name] = []byte(value)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("secret %q has no keys", secretName)
	}

	return keys, nil
}

func (c *SecretsContext) GetIntegrationKeys(integrationName string) (map[string][]byte, error) {
	name := strings.TrimSpace(integrationName)
	if name == "" {
		return nil, fmt.Errorf("integration name is required")
	}

	integration, err := models.FindIntegrationByName(c.tx, c.organizationID, name)
	if err != nil {
		return nil, err
	}

	if integration.State != models.IntegrationStateReady {
		return nil, fmt.Errorf("integration %q is not ready", integration.InstallationName)
	}

	integrationImpl, err := c.registry.GetIntegration(integration.AppName)
	if err != nil {
		return nil, err
	}

	provider, ok := registry.UnwrapIntegration(integrationImpl).(core.IntegrationSecretProvider)
	if !ok {
		return nil, fmt.Errorf("integration %q does not provide secrets", integration.InstallationName)
	}

	secretCtx := core.IntegrationSecretContext{
		Logger:      logging.ForIntegration(*integration),
		HTTP:        c.registry.HTTPContextInTransaction(c.tx),
		Integration: NewIntegrationContext(c.tx, nil, integration, c.encryptor, c.registry, nil),
	}

	return provider.ResolveSecrets(secretCtx)
}

func (c *SecretsContext) decryptSecretData(secret *models.Secret) (map[string]string, error) {
	plain, err := c.encryptor.Decrypt(context.Background(), secret.Data, []byte(secret.Name))
	if err != nil {
		return nil, err
	}

	var data map[string]string
	if len(plain) == 0 {
		return make(map[string]string), nil
	}
	if err := json.Unmarshal(plain, &data); err != nil {
		return nil, err
	}
	return data, nil
}
