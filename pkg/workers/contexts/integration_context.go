package contexts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type IntegrationContext struct {
	tx              *gorm.DB
	node            *models.WorkflowNode
	appInstallation *models.AppInstallation
	encryptor       crypto.Encryptor
	registry        *registry.Registry
}

func NewIntegrationContext(tx *gorm.DB, node *models.WorkflowNode, installation *models.AppInstallation, encryptor crypto.Encryptor, registry *registry.Registry) *IntegrationContext {
	return &IntegrationContext{
		tx:              tx,
		node:            node,
		appInstallation: installation,
		encryptor:       encryptor,
		registry:        registry,
	}
}

func (c *IntegrationContext) ID() uuid.UUID {
	return c.appInstallation.ID
}

func (c *IntegrationContext) RequestWebhook(configuration any) error {
	integration, err := c.registry.GetIntegration(c.appInstallation.AppName)
	if err != nil {
		return err
	}

	webhooks, err := models.ListAppInstallationWebhooks(c.tx, c.appInstallation.ID)
	if err != nil {
		return fmt.Errorf("Failed to list webhooks: %v", err)
	}

	for _, hook := range webhooks {
		ok, err := integration.CompareWebhookConfig(hook.Configuration.Data(), configuration)
		if err != nil {
			return err
		}

		if ok {
			c.node.WebhookID = &hook.ID
			return nil
		}
	}

	return c.createWebhook(configuration)
}

func (c *IntegrationContext) createWebhook(configuration any) error {
	webhookID := uuid.New()
	_, encryptedKey, err := crypto.NewRandomKey(context.Background(), c.encryptor, webhookID.String())
	if err != nil {
		return fmt.Errorf("error generating key for new webhook: %v", err)
	}

	now := time.Now()
	webhook := models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStatePending,
		Secret:            encryptedKey,
		Configuration:     datatypes.NewJSONType(configuration),
		AppInstallationID: &c.appInstallation.ID,
		CreatedAt:         &now,
	}

	err = c.tx.Create(&webhook).Error
	if err != nil {
		return err
	}

	c.node.WebhookID = &webhookID
	return nil
}

func (c *IntegrationContext) ScheduleResync(interval time.Duration) error {
	if interval < time.Second {
		return fmt.Errorf("interval must be bigger than 1s")
	}

	err := c.completeCurrentRequestForInstallation()
	if err != nil {
		return err
	}

	runAt := time.Now().Add(interval)
	return c.appInstallation.CreateSyncRequest(c.tx, &runAt)
}

func (c *IntegrationContext) completeCurrentRequestForInstallation() error {
	request, err := models.FindPendingRequestForAppInstallation(c.tx, c.appInstallation.ID)
	if err == nil {
		return request.Complete(c.tx)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return err
}

func (c *IntegrationContext) GetConfig(name string) ([]byte, error) {
	config := c.appInstallation.Configuration.Data()
	v, ok := config[name]
	if !ok {
		return nil, fmt.Errorf("config %s not found", name)
	}

	integration, err := c.registry.GetIntegration(c.appInstallation.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to get integration %s: %w", c.appInstallation.AppName, err)
	}

	configDef, err := findConfigDef(integration.Configuration(), name)
	if err != nil {
		return nil, fmt.Errorf("failed to find config %s: %w", name, err)
	}

	if configDef.Type != configuration.FieldTypeString && configDef.Type != configuration.FieldTypeSelect && configDef.Type != configuration.FieldTypeText {
		return nil, fmt.Errorf("config %s is not of type: [string, select, text]", name)
	}

	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("config %s is not a string", name)
	}

	if !configDef.Sensitive {
		return []byte(s), nil
	}

	decoded, err := base64.StdEncoding.DecodeString(string(s))
	if err != nil {
		return nil, err
	}

	return c.encryptor.Decrypt(context.Background(), []byte(decoded), []byte(c.appInstallation.ID.String()))
}

func findConfigDef(configs []configuration.Field, name string) (configuration.Field, error) {
	for _, config := range configs {
		if config.Name == name {
			return config, nil
		}
	}

	return configuration.Field{}, fmt.Errorf("config %s not found", name)
}

func (c *IntegrationContext) GetMetadata() any {
	return c.appInstallation.Metadata.Data()
}

func (c *IntegrationContext) SetMetadata(value any) {
	b, err := json.Marshal(value)
	if err != nil {
		return
	}

	var v map[string]any
	err = json.Unmarshal(b, &v)
	if err != nil {
		return
	}

	c.appInstallation.Metadata = datatypes.NewJSONType(v)
}

func (c *IntegrationContext) GetState() string {
	return c.appInstallation.State
}

func (c *IntegrationContext) SetState(state, stateDescription string) {
	c.appInstallation.State = state
	c.appInstallation.StateDescription = stateDescription
}

func (c *IntegrationContext) SetSecret(name string, value []byte) error {
	now := time.Now()

	// Encrypt the secret value using the installation ID as associated data
	encryptedValue, err := c.encryptor.Encrypt(
		context.Background(),
		value,
		[]byte(c.appInstallation.ID.String()),
	)
	if err != nil {
		return err
	}

	var secret models.AppInstallationSecret
	err = c.tx.
		Where("installation_id = ?", c.appInstallation.ID).
		Where("name = ?", name).
		First(&secret).
		Error

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		secret = models.AppInstallationSecret{
			OrganizationID: c.appInstallation.OrganizationID,
			InstallationID: c.appInstallation.ID,
			Name:           name,
			Value:          encryptedValue,
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}

		return c.tx.Create(&secret).Error
	}

	secret.Value = encryptedValue
	secret.UpdatedAt = &now

	return c.tx.Save(&secret).Error
}

func (c *IntegrationContext) GetSecrets() ([]core.IntegrationSecret, error) {
	var fromDB []models.AppInstallationSecret
	err := c.tx.
		Where("installation_id = ?", c.appInstallation.ID).
		Find(&fromDB).
		Error

	if err != nil {
		return nil, err
	}

	var secrets []core.IntegrationSecret
	for _, secret := range fromDB {
		decryptedValue, err := c.encryptor.Decrypt(
			context.Background(),
			secret.Value,
			[]byte(c.appInstallation.ID.String()),
		)

		if err != nil {
			return nil, err
		}

		secrets = append(secrets, core.IntegrationSecret{
			Name:  secret.Name,
			Value: decryptedValue,
		})
	}

	return secrets, nil
}

func (c *IntegrationContext) NewBrowserAction(action core.BrowserAction) {
	d := datatypes.NewJSONType(models.BrowserAction{
		URL:         action.URL,
		Method:      action.Method,
		FormFields:  action.FormFields,
		Description: action.Description,
	})

	c.appInstallation.BrowserAction = &d
}

func (c *IntegrationContext) RemoveBrowserAction() {
	c.appInstallation.BrowserAction = nil
}

func (c *IntegrationContext) Subscribe(configuration any) (*uuid.UUID, error) {
	subscription, err := models.CreateAppSubscriptionInTransaction(c.tx, c.node, c.appInstallation, configuration)
	if err != nil {
		return nil, err
	}

	return &subscription.ID, nil
}

func (c *IntegrationContext) ListSubscriptions() ([]core.IntegrationSubscriptionContext, error) {
	subscriptions, err := models.ListAppSubscriptions(c.tx, c.appInstallation.ID)
	if err != nil {
		return nil, err
	}

	contexts := []core.IntegrationSubscriptionContext{}
	for _, subscription := range subscriptions {
		node, err := models.FindWorkflowNode(c.tx, subscription.WorkflowID, subscription.NodeID)
		if err != nil {
			return nil, err
		}

		contexts = append(contexts, NewIntegrationSubscriptionContext(
			c.tx,
			c.registry,
			&subscription,
			node,
			c.appInstallation,
			c,
		))
	}

	return contexts, nil
}
