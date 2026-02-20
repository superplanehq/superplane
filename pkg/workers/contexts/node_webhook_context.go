package contexts

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type NodeWebhookContext struct {
	tx        *gorm.DB
	ctx       context.Context
	encryptor crypto.Encryptor
	node      *models.CanvasNode
	baseURL   string
}

func NewNodeWebhookContext(ctx context.Context, tx *gorm.DB, encryptor crypto.Encryptor, node *models.CanvasNode, baseURL string) *NodeWebhookContext {
	return &NodeWebhookContext{
		tx:        tx,
		ctx:       ctx,
		node:      node,
		encryptor: encryptor,
		baseURL:   baseURL,
	}
}

func (c *NodeWebhookContext) GetSecret() ([]byte, error) {
	if c.node.WebhookID == nil {
		return nil, fmt.Errorf("node does not have a webhook")
	}

	webhook, err := models.FindWebhookInTransaction(c.tx, *c.node.WebhookID)
	if err != nil {
		return nil, err
	}

	return c.encryptor.Decrypt(c.ctx, webhook.Secret, []byte(webhook.ID.String()))
}

func (c *NodeWebhookContext) SetSecret(secret []byte) error {
	if c.node.WebhookID == nil {
		return fmt.Errorf("node does not have a webhook")
	}

	webhook, err := models.FindWebhookInTransaction(c.tx, *c.node.WebhookID)
	if err != nil {
		return err
	}

	encrypted, err := c.encryptor.Encrypt(c.ctx, secret, []byte(webhook.ID.String()))
	if err != nil {
		return err
	}

	webhook.Secret = encrypted
	return c.tx.Model(webhook).Update("secret", webhook.Secret).Error
}

func (c *NodeWebhookContext) ResetSecret() ([]byte, []byte, error) {
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

func (c *NodeWebhookContext) Setup() (string, error) {
	webhook, err := c.findOrCreateWebhook()
	if err != nil {
		return "", fmt.Errorf("failed to find or create webhook: %w", err)
	}

	c.node.WebhookID = &webhook.ID
	// Must include /api/v1 to match the public route; WebhookContext.GetURL uses the same pattern
	return fmt.Sprintf("%s/api/v1/webhooks/%s", c.GetBaseURL(), webhook.ID.String()), nil
}

func (c *NodeWebhookContext) findOrCreateWebhook() (*models.Webhook, error) {
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

	err = c.tx.Create(&webhook).Error
	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func (c *NodeWebhookContext) GetBaseURL() string {
	return c.baseURL
}
