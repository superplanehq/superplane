package contexts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type WebhookContext struct {
	tx        *gorm.DB
	webhook   *models.Webhook
	encryptor crypto.Encryptor
	baseURL   string
}

func NewWebhookContext(tx *gorm.DB, webhook *models.Webhook, encryptor crypto.Encryptor, baseURL string) *WebhookContext {
	return &WebhookContext{
		tx:        tx,
		webhook:   webhook,
		encryptor: encryptor,
		baseURL:   baseURL,
	}
}

func (c *WebhookContext) GetID() string {
	return c.webhook.ID.String()
}

func (c *WebhookContext) GetURL() string {
	return fmt.Sprintf("%s/api/v1/webhooks/%s", c.baseURL, c.webhook.ID)
}

func (c *WebhookContext) CallbackHasActiveNodes(webhookID string) (bool, error) {
	id, err := uuid.Parse(webhookID)
	if err != nil {
		return false, fmt.Errorf("invalid webhook id: %w", err)
	}

	nodes, err := models.FindActiveWebhookNodesInTransaction(c.tx, id)
	if err != nil {
		return false, err
	}

	return len(nodes) > 0, nil
}

func (c *WebhookContext) GetSecret() ([]byte, error) {
	secret, err := c.encryptor.Decrypt(context.Background(), c.webhook.Secret, []byte(c.webhook.ID.String()))
	if err != nil {
		return nil, err
	}

	return secret, nil
}

func (c *WebhookContext) GetMetadata() any {
	return c.webhook.Metadata.Data()
}

func (c *WebhookContext) GetConfiguration() any {
	return c.webhook.Configuration.Data()
}

func (c *WebhookContext) SetSecret(secret []byte) error {
	encrypted, err := c.encryptor.Encrypt(context.Background(), secret, []byte(c.webhook.ID.String()))
	if err != nil {
		return err
	}

	c.webhook.Secret = encrypted
	return c.tx.
		Model(c.webhook).
		Update("secret", c.webhook.Secret).
		Error
}
