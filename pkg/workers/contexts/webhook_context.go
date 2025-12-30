package contexts

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WebhookContext struct {
	tx        *gorm.DB
	ctx       context.Context
	encryptor crypto.Encryptor
	node      *models.WorkflowNode
	baseURL   string
}

func NewWebhookContext(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, node *models.WorkflowNode, baseURL string) *WebhookContext {
	return &WebhookContext{
		tx:        tx,
		ctx:       ctx,
		node:      node,
		encryptor: encryptor,
		baseURL:   baseURL,
	}
}

func (c *WebhookContext) GetSecret() ([]byte, error) {
	if c.node.WebhookID == nil {
		return nil, fmt.Errorf("node does not have a webhook")
	}

	webhook, err := models.FindWebhookInTransaction(c.tx, *c.node.WebhookID)
	if err != nil {
		return nil, err
	}

	return c.encryptor.Decrypt(c.ctx, webhook.Secret, []byte(webhook.ID.String()))
}

func (c *WebhookContext) ResetSecret() ([]byte, []byte, error) {
	if c.node.WebhookID == nil {
		return nil, nil, fmt.Errorf("node does not have a webhook")
	}

	plainKey, encryptedKey, err := crypto.NewRandomKey(c.ctx, c.encryptor, c.node.WebhookID.String())
	if err != nil {
		return nil, nil, fmt.Errorf("error generating key for new webhook: %v", err)
	}

	webhook, err := models.FindWebhookInTransaction(c.tx, *c.node.WebhookID)
	if err != nil {
		return nil, nil, fmt.Errorf("error finding webhook: %v", err)
	}

	webhook.Secret = encryptedKey
	err = c.tx.Save(webhook).Error
	if err != nil {
		return nil, nil, fmt.Errorf("error saving webhook: %v", err)
	}

	return []byte(plainKey), encryptedKey, nil
}

func (c *WebhookContext) Setup(options *core.WebhookSetupOptions) (*uuid.UUID, error) {
	webhook, err := c.findOrCreateWebhook(options)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create webhook: %w", err)
	}

	c.node.WebhookID = &webhook.ID
	return &webhook.ID, nil
}

func (c *WebhookContext) findOrCreateWebhook(options *core.WebhookSetupOptions) (*models.Webhook, error) {
	//
	// If webhook already exists, just return it
	//
	if c.node.WebhookID != nil {
		return models.FindWebhookInTransaction(c.tx, *c.node.WebhookID)
	}

	//
	// Otherwise, create it.
	//
	webhookID := uuid.New()
	_, encryptedKey, err := crypto.NewRandomKey(c.ctx, c.encryptor, webhookID.String())
	if err != nil {
		return nil, fmt.Errorf("error generating key for new webhook: %v", err)
	}

	now := time.Now()
	webhook := models.Webhook{
		ID:        webhookID,
		State:     models.WebhookStatePending,
		Secret:    encryptedKey,
		CreatedAt: &now,
	}

	if options == nil {
		err = c.tx.Create(&webhook).Error
		if err != nil {
			return nil, err
		}

		return &webhook, nil
	}

	if options.IntegrationID != nil {
		webhook.IntegrationID = options.IntegrationID
	}

	if options.Resource != nil {
		webhook.Resource = datatypes.NewJSONType(models.WebhookResource{
			ID:   options.Resource.Id(),
			Name: options.Resource.Name(),
			Type: options.Resource.Type(),
		})
	}

	if options.Configuration != nil {
		webhook.Configuration = datatypes.NewJSONType(options.Configuration)
	}

	err = c.tx.Create(&webhook).Error
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (c *WebhookContext) GetBaseURL() string {
	return c.baseURL
}
