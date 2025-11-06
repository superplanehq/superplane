package contexts

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/triggers"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WebhookContext struct {
	tx        *gorm.DB
	ctx       context.Context
	encryptor crypto.Encryptor
	node      *models.WorkflowNode
}

func NewWebhookContext(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, node *models.WorkflowNode) triggers.WebhookContext {
	return &WebhookContext{
		tx:        tx,
		ctx:       ctx,
		node:      node,
		encryptor: encryptor,
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

func (c *WebhookContext) Setup(options *triggers.WebhookSetupOptions) error {
	webhook, err := c.findOrCreateWebhook(options)
	if err != nil {
		return fmt.Errorf("failed to find or create webhook: %w", err)
	}

	c.node.WebhookID = &webhook.ID
	return nil
}

func (c *WebhookContext) findOrCreateWebhook(options *triggers.WebhookSetupOptions) (*models.Webhook, error) {
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
