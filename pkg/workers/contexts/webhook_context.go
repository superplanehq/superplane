package contexts

import (
	"context"
	"fmt"
	"log"

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

func NewWebhookContext(tx *gorm.DB, ctx context.Context, encryptor crypto.Encryptor, node *models.WorkflowNode) triggers.WebhookContext {
	return &WebhookContext{
		tx:        tx,
		ctx:       ctx,
		node:      node,
		encryptor: encryptor,
	}
}

func (c *WebhookContext) GetSecret() ([]byte, error) {
	metadata := c.node.Metadata.Data()
	webhookReference := metadata["webhook"].(string)

	//
	// If webhook already exists, just return it
	//
	if webhookReference == "" {
		return []byte{}, fmt.Errorf("webhook not found")
	}

	webhookID, err := uuid.Parse(webhookReference)
	if err != nil {
		return nil, err
	}

	webhook, err := models.FindWebhook(webhookID)
	if err != nil {
		return nil, err
	}

	return c.encryptor.Decrypt(c.ctx, webhook.Secret, []byte(webhook.ID.String()))
}

func (c *WebhookContext) Setup(actionName string) error {
	webhook, err := c.findOrCreateWebhook()
	if err != nil {
		return fmt.Errorf("failed to find or create webhook: %w", err)
	}

	handlers, err := models.FindWebhookHandlers(webhook.ID.String())
	if err != nil {
		return fmt.Errorf("failed to find webhook handlers: %w", err)
	}

	//
	// If handler already exists, no need to create it again
	//
	handler := c.findHandler(handlers, actionName)
	if handler != nil {
		return nil
	}

	return models.CreateWebhookHandler(c.tx, c.node.WorkflowID, c.node.NodeID, webhook.ID, models.WebhookHandlerSpec{
		InvokeAction: &models.InvokeAction{
			ActionName: actionName,
		},
	})
}

func (c *WebhookContext) findOrCreateWebhook() (*models.Webhook, error) {
	metadata := c.node.Metadata.Data()
	webhookReference, ok := metadata["webhook"].(string)

	//
	// If webhook already exists, just return it
	//
	if ok && webhookReference != "" {
		webhookID, err := uuid.Parse(webhookReference)
		if err != nil {
			return nil, err
		}

		return models.FindWebhook(webhookID)
	}

	//
	// Otherwise, create it.
	// TODO: how do we give the plain key back to the user?
	//
	webhookID := uuid.New()
	plainText, encryptedKey, err := crypto.NewRandomKey(c.ctx, c.encryptor, webhookID.String())
	if err != nil {
		return nil, fmt.Errorf("error generating key for new webhook: %v", err)
	}

	webhook := models.Webhook{
		ID:     webhookID,
		Secret: encryptedKey,
	}

	err = c.tx.Create(&webhook).Error
	if err != nil {
		return nil, err
	}

	log.Printf("New webhook created: %s", webhookID.String())
	log.Printf("New webhook key: %s", plainText)

	//
	// Save webhook reference in node metadata.
	//
	metadata = map[string]any{
		"webhook": webhook.ID.String(),
	}

	c.node.Metadata = datatypes.NewJSONType(metadata)
	return &webhook, nil
}

func (c *WebhookContext) findHandler(handlers []models.WebhookHandler, actionName string) *models.WebhookHandler {
	for _, handler := range handlers {
		spec := handler.Spec.Data()
		if handler.NodeID == c.node.NodeID && spec.InvokeAction != nil && spec.InvokeAction.ActionName == actionName {
			return &handler
		}
	}

	return nil
}
