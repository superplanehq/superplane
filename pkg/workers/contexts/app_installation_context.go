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

type AppInstallationContext struct {
	tx              *gorm.DB
	node            *models.WorkflowNode
	appInstallation *models.AppInstallation
	encryptor       crypto.Encryptor
	registry        *registry.Registry
}

func NewAppInstallationContext(tx *gorm.DB, node *models.WorkflowNode, installation *models.AppInstallation, encryptor crypto.Encryptor, registry *registry.Registry) *AppInstallationContext {
	return &AppInstallationContext{
		tx:              tx,
		node:            node,
		appInstallation: installation,
		encryptor:       encryptor,
		registry:        registry,
	}
}

func (c *AppInstallationContext) ID() uuid.UUID {
	return c.appInstallation.ID
}

func (c *AppInstallationContext) RequestWebhook(configuration any) error {
	app, err := c.registry.GetApplication(c.appInstallation.AppName)
	if err != nil {
		return err
	}

	return app.RequestWebhook(c, configuration)
}

func (c *AppInstallationContext) AssociateWebhook(id uuid.UUID) {
	c.node.WebhookID = &id
}

func (c *AppInstallationContext) CreateWebhook(configuration any) error {
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

func (c *AppInstallationContext) ListWebhooks() ([]core.Webhook, error) {
	webhooks, err := models.ListAppInstallationWebhooks(c.tx, c.appInstallation.ID)
	if err != nil {
		return nil, err
	}

	hooks := []core.Webhook{}
	for _, webhook := range webhooks {
		hooks = append(hooks, core.Webhook{
			ID:            webhook.ID,
			Configuration: webhook.Configuration.Data(),
		})
	}

	return hooks, nil
}

func (c *AppInstallationContext) GetConfig(name string) ([]byte, error) {
	config := c.appInstallation.Configuration.Data()
	v, ok := config[name]
	if !ok {
		return nil, fmt.Errorf("config %s not found", name)
	}

	app, err := c.registry.GetApplication(c.appInstallation.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to get app %s: %w", c.appInstallation.AppName, err)
	}

	configDef, err := findConfigDef(app.Configuration(), name)
	if err != nil {
		return nil, fmt.Errorf("failed to find config %s: %w", name, err)
	}

	if configDef.Type != configuration.FieldTypeString {
		return nil, fmt.Errorf("config %s is not a string", name)
	}

	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("config %s is not a string", name)
	}

	if !configDef.Sensitive {
		return []byte(s), nil
	}

	b64, err := c.encryptor.Decrypt(context.Background(), []byte(s), []byte(c.appInstallation.ID.String()))
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(string(b64))
}

func findConfigDef(configs []configuration.Field, name string) (configuration.Field, error) {
	for _, config := range configs {
		if config.Name == name {
			return config, nil
		}
	}

	return configuration.Field{}, fmt.Errorf("config %s not found", name)
}

func (m *AppInstallationContext) GetMetadata() any {
	return m.appInstallation.Metadata.Data()
}

func (m *AppInstallationContext) SetMetadata(value any) {
	b, err := json.Marshal(value)
	if err != nil {
		return
	}

	var v map[string]any
	err = json.Unmarshal(b, &v)
	if err != nil {
		return
	}

	m.appInstallation.Metadata = datatypes.NewJSONType(v)
}

func (m *AppInstallationContext) GetState() string {
	return m.appInstallation.State
}

func (m *AppInstallationContext) SetState(value string) {
	m.appInstallation.State = value
}

func (m *AppInstallationContext) SetSecret(name string, value []byte) error {
	now := time.Now()

	// Encrypt the secret value using the installation ID as associated data
	encryptedValue, err := m.encryptor.Encrypt(
		context.Background(),
		value,
		[]byte(m.appInstallation.ID.String()),
	)
	if err != nil {
		return err
	}

	var secret models.AppInstallationSecret
	err = m.tx.
		Where("installation_id = ?", m.appInstallation.ID).
		Where("name = ?", name).
		First(&secret).
		Error

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		secret = models.AppInstallationSecret{
			OrganizationID: m.appInstallation.OrganizationID,
			InstallationID: m.appInstallation.ID,
			Name:           name,
			Value:          encryptedValue,
			CreatedAt:      &now,
			UpdatedAt:      &now,
		}

		return m.tx.Create(&secret).Error
	}

	secret.Value = encryptedValue
	secret.UpdatedAt = &now

	return m.tx.Save(&secret).Error
}

func (m *AppInstallationContext) GetSecrets() ([]core.InstallationSecret, error) {
	var fromDB []models.AppInstallationSecret
	err := m.tx.
		Where("installation_id = ?", m.appInstallation.ID).
		Find(&fromDB).
		Error

	if err != nil {
		return nil, err
	}

	var secrets []core.InstallationSecret
	for _, secret := range fromDB {
		decryptedValue, err := m.encryptor.Decrypt(
			context.Background(),
			secret.Value,
			[]byte(m.appInstallation.ID.String()),
		)

		if err != nil {
			return nil, err
		}

		secrets = append(secrets, core.InstallationSecret{
			Name:  secret.Name,
			Value: decryptedValue,
		})
	}

	return secrets, nil
}

func (m *AppInstallationContext) NewBrowserAction(action core.BrowserAction) {
	d := datatypes.NewJSONType(models.BrowserAction{
		URL:         action.URL,
		Method:      action.Method,
		FormFields:  action.FormFields,
		Description: action.Description,
	})

	m.appInstallation.BrowserAction = &d
}

func (m *AppInstallationContext) RemoveBrowserAction() {
	m.appInstallation.BrowserAction = nil
}
