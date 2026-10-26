package contexts

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryption"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	runnerapi "github.com/superplanehq/superplane/pkg/runners/api"
	"gorm.io/gorm"
)

/*
 * SecretsContext implements core.SecretsContext,
 * resolving organization secret key values for component execution.
 */
type SecretsContext struct {
	tx                     *gorm.DB
	registry               *registry.Registry
	encryptor              crypto.Encryptor
	organizationID         uuid.UUID
	refreshes              []runnerapi.CredentialRefresh
	deferRunnerCredentials bool
	deferSecretValues      bool
}

func (c *SecretsContext) DeferRunnerSecretValues() { c.deferSecretValues = true }

func NewSecretsContext(tx *gorm.DB, reg *registry.Registry, organizationID uuid.UUID, encryptor crypto.Encryptor) *SecretsContext {
	return &SecretsContext{
		tx:             tx,
		encryptor:      encryption.ForDB(tx, encryptor),
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
	if c.deferSecretValues {
		val = runnerSecretPlaceholder(secret.ID, keyName)
	}
	c.recordSecretRefresh(secret, keyName, val)

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
		if c.deferSecretValues {
			value = runnerSecretPlaceholder(secret.ID, name)
		}
		keys[name] = []byte(value)
		c.recordSecretRefresh(secret, name, value)
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("secret %q has no keys", secretName)
	}

	return keys, nil
}

func (c *SecretsContext) GetIntegrationSecrets(integrationName string) (core.IntegrationSecrets, error) {
	name := strings.TrimSpace(integrationName)
	if name == "" {
		return core.IntegrationSecrets{}, fmt.Errorf("integration name is required")
	}

	integration, err := models.FindIntegrationByName(c.tx, c.organizationID, name)
	if err != nil {
		return core.IntegrationSecrets{}, err
	}

	if integration.State != models.IntegrationStateReady {
		return core.IntegrationSecrets{}, fmt.Errorf("integration %q is not ready", integration.InstallationName)
	}

	integrationImpl, err := c.registry.GetIntegration(integration.AppName)
	if err != nil {
		return core.IntegrationSecrets{}, err
	}

	provider, ok := registry.UnwrapIntegration(integrationImpl).(core.IntegrationSecretProvider)
	if !ok {
		return core.IntegrationSecrets{}, fmt.Errorf("integration %q does not provide secrets", integration.InstallationName)
	}

	secretCtx := core.IntegrationSecretContext{
		Logger:      logging.ForIntegration(*integration),
		HTTP:        c.registry.HTTPContextInTransaction(c.tx),
		Integration: NewIntegrationContext(c.tx, nil, integration, c.encryptor, c.registry, nil),
	}

	var resolved core.IntegrationSecrets
	if descriptor, ok := registry.UnwrapIntegration(integrationImpl).(core.IntegrationRunnerSecretDescriptor); c.deferRunnerCredentials && ok {
		resolved, err = descriptor.DescribeRunnerSecrets(secretCtx)
	} else {
		resolved, err = provider.ResolveSecrets(secretCtx)
	}
	if err != nil {
		return resolved, err
	}
	permittedScope, err := RunnerIntegrationScope(c.tx, integration)
	if err != nil {
		return resolved, err
	}
	keyIdentity, err := IntegrationCredentialKeyIdentity(c.tx, c.encryptor, integration)
	if err != nil {
		return resolved, err
	}
	for key, value := range resolved.Values {
		if c.deferRunnerCredentials {
			value = []byte(runnerSecretPlaceholder(integration.ID, key))
			resolved.Values[key] = value
		}
		c.refreshes = append(c.refreshes, runnerapi.CredentialRefresh{Version: 1, Kind: "integration", OrganizationID: c.organizationID.String(), ResourceID: integration.ID.String(), Recipient: integration.AppName, Scope: map[string]string{"key": key, "identity": integrationCredentialIdentity(integration), "permitted_scope": permittedScope, "key_identity": keyIdentity}, Current: string(value)})
	}
	return resolved, nil
}

func runnerSecretPlaceholder(resourceID uuid.UUID, key string) string {
	return "pending-runner-credential-" + resourceID.String() + "-" + key
}

func (c *SecretsContext) recordSecretRefresh(secret *models.Secret, key, value string) {
	c.refreshes = append(c.refreshes, runnerapi.CredentialRefresh{Version: 1, Kind: "secret", OrganizationID: c.organizationID.String(), ResourceID: secret.ID.String(), Recipient: "organization", Scope: map[string]string{"key": key}, Current: value})
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
