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
	tx          *gorm.DB
	node        *models.CanvasNode
	integration *models.Integration
	encryptor   crypto.Encryptor
	registry    *registry.Registry
}

func NewIntegrationContext(tx *gorm.DB, node *models.CanvasNode, integration *models.Integration, encryptor crypto.Encryptor, registry *registry.Registry) *IntegrationContext {
	return &IntegrationContext{
		tx:          tx,
		node:        node,
		integration: integration,
		encryptor:   encryptor,
		registry:    registry,
	}
}

func (c *IntegrationContext) ID() uuid.UUID {
	return c.integration.ID
}

func (c *IntegrationContext) RequestWebhook(configuration any) error {
	handler, err := c.registry.GetWebhookHandler(c.integration.AppName)
	if err != nil {
		return err
	}

	if err := c.replaceMismatchedWebhook(configuration, handler); err != nil {
		return err
	}

	webhooks, err := models.ListIntegrationWebhooks(c.tx, c.integration.ID)
	if err != nil {
		return fmt.Errorf("Failed to list webhooks: %v", err)
	}

	for _, hook := range webhooks {
		ok, err := handler.CompareConfig(hook.Configuration.Data(), configuration)
		if err != nil {
			return err
		}

		if ok {
			if err := c.mergeWebhookConfiguration(handler, &hook, configuration); err != nil {
				return err
			}

			c.node.WebhookID = &hook.ID
			return nil
		}
	}

	return c.createWebhook(configuration)
}

func (c *IntegrationContext) replaceMismatchedWebhook(configuration any, handler core.WebhookHandler) error {
	if c.node == nil || c.node.WebhookID == nil {
		return nil
	}

	webhook, err := models.FindWebhookInTransaction(c.tx, *c.node.WebhookID)
	if err != nil {
		return err
	}

	matches, err := handler.CompareConfig(webhook.Configuration.Data(), configuration)
	if err != nil {
		return err
	}

	if matches {
		return nil
	}

	c.node.WebhookID = nil

	nodes, err := models.FindWebhookNodesInTransaction(c.tx, webhook.ID)
	if err != nil {
		return err
	}

	if len(nodes) > 1 {
		return nil
	}

	return c.tx.Delete(webhook).Error
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
		AppInstallationID: &c.integration.ID,
		CreatedAt:         &now,
	}

	err = c.tx.Create(&webhook).Error
	if err != nil {
		return err
	}

	c.node.WebhookID = &webhookID
	return nil
}

func (c *IntegrationContext) mergeWebhookConfiguration(
	handler core.WebhookHandler,
	webhook *models.Webhook,
	configuration any,
) error {
	mergedConfiguration, changed, err := handler.Merge(webhook.Configuration.Data(), configuration)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	webhook.Configuration = datatypes.NewJSONType(mergedConfiguration)
	webhook.State = models.WebhookStatePending
	webhook.RetryCount = 0

	return c.tx.Model(webhook).Updates(map[string]any{
		"configuration": webhook.Configuration,
		"state":         webhook.State,
		"retry_count":   webhook.RetryCount,
		"updated_at":    time.Now(),
	}).Error
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
	return c.integration.CreateSyncRequest(c.tx, &runAt)
}

func (c *IntegrationContext) ScheduleActionCall(actionName string, parameters any, interval time.Duration) error {
	if interval < time.Second {
		return fmt.Errorf("interval must be bigger than 1s")
	}

	runAt := time.Now().Add(interval)
	return c.integration.CreateActionRequest(c.tx, actionName, parameters, &runAt)
}

func (c *IntegrationContext) completeCurrentRequestForInstallation() error {
	request, err := models.FindPendingRequestForIntegration(c.tx, c.integration.ID)
	if err == nil {
		return request.Complete(c.tx)
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}

	return err
}

func (c *IntegrationContext) GetConfig(name string) ([]byte, error) {
	config := c.integration.Configuration.Data()
	v, ok := config[name]
	if !ok {
		return nil, fmt.Errorf("config %s not found", name)
	}

	impl, err := c.registry.GetIntegration(c.integration.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to get integration %s: %w", c.integration.AppName, err)
	}

	configDef, err := findConfigDef(impl.Configuration(), name)
	if err != nil {
		return nil, fmt.Errorf("failed to find config %s: %w", name, err)
	}

	if configDef.Type != configuration.FieldTypeString && configDef.Type != configuration.FieldTypeSelect {
		return nil, fmt.Errorf("config %s is not of type: [string, select]", name)
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

	return c.encryptor.Decrypt(context.Background(), []byte(decoded), []byte(c.integration.ID.String()))
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
	return c.integration.Metadata.Data()
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

	c.integration.Metadata = datatypes.NewJSONType(v)
}

func (c *IntegrationContext) GetState() string {
	return c.integration.State
}

func (c *IntegrationContext) Ready() {
	c.integration.State = models.IntegrationStateReady
	c.integration.StateDescription = ""
}

func (c *IntegrationContext) Error(message string) {
	c.integration.State = models.IntegrationStateError
	c.integration.StateDescription = message
}

func (c *IntegrationContext) SetSecret(name string, value []byte) error {
	now := time.Now()

	// Encrypt the secret value using the installation ID as associated data
	encryptedValue, err := c.encryptor.Encrypt(
		context.Background(),
		value,
		[]byte(c.integration.ID.String()),
	)
	if err != nil {
		return err
	}

	var secret models.IntegrationSecret
	err = c.tx.
		Where("installation_id = ?", c.integration.ID).
		Where("name = ?", name).
		First(&secret).
		Error

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		secret = models.IntegrationSecret{
			OrganizationID: c.integration.OrganizationID,
			InstallationID: c.integration.ID,
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
	var fromDB []models.IntegrationSecret
	err := c.tx.
		Where("installation_id = ?", c.integration.ID).
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
			[]byte(c.integration.ID.String()),
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

	c.integration.BrowserAction = &d
}

func (c *IntegrationContext) RemoveBrowserAction() {
	c.integration.BrowserAction = nil
}

func (c *IntegrationContext) Subscribe(configuration any) (*uuid.UUID, error) {
	subscription, err := models.CreateIntegrationSubscriptionInTransaction(c.tx, c.node, c.integration, configuration)
	if err != nil {
		return nil, err
	}

	return &subscription.ID, nil
}

func (c *IntegrationContext) ListSubscriptions() ([]core.IntegrationSubscriptionContext, error) {
	subscriptions, err := models.ListIntegrationSubscriptions(c.tx, c.integration.ID)
	if err != nil {
		return nil, err
	}

	contexts := []core.IntegrationSubscriptionContext{}
	for _, subscription := range subscriptions {
		node, err := models.FindCanvasNode(c.tx, subscription.WorkflowID, subscription.NodeID)
		if err != nil {
			return nil, err
		}

		contexts = append(contexts, NewIntegrationSubscriptionContext(
			c.tx,
			c.registry,
			&subscription,
			node,
			c.integration,
			c,
		))
	}

	return contexts, nil
}

// FindSubscription finds the first subscription matching the predicate.
// Note: This loads all subscriptions via ListSubscriptions() and iterates through them.
// For installations with many subscriptions, this may be inefficient. Consider adding
// an index-based lookup if performance becomes an issue.
func (c *IntegrationContext) FindSubscription(predicate func(core.IntegrationSubscriptionContext) bool) (core.IntegrationSubscriptionContext, error) {
	subscriptions, err := c.ListSubscriptions()
	if err != nil {
		return nil, err
	}

	for _, subscription := range subscriptions {
		if predicate(subscription) {
			return subscription, nil
		}
	}

	return nil, nil
}
