package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Webhook struct {
	ID     uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Secret []byte
}

type WebhookHandler struct {
	ID         uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	WebhookID  uuid.UUID
	WorkflowID uuid.UUID
	NodeID     string
	Spec       datatypes.JSONType[WebhookHandlerSpec]
}

type WebhookHandlerSpec struct {
	InvokeAction *InvokeAction `json:"invoke_action,omitempty"`
}

func FindWebhook(id uuid.UUID) (*Webhook, error) {
	var webhook Webhook
	err := database.Conn().
		First(&webhook, id).
		Error

	if err != nil {
		return nil, err
	}

	return &webhook, nil
}

func FindWebhookHandlers(webhookID string) ([]WebhookHandler, error) {
	var handlers []WebhookHandler
	err := database.Conn().
		Where("webhook_id = ?", webhookID).
		Find(&handlers).
		Error

	if err != nil {
		return nil, err
	}

	return handlers, nil
}

func CreateWebhookHandler(tx *gorm.DB, workflowID uuid.UUID, nodeID string, webhookID uuid.UUID, spec WebhookHandlerSpec) error {
	return tx.
		Create(&WebhookHandler{
			WebhookID:  webhookID,
			WorkflowID: workflowID,
			NodeID:     nodeID,
			Spec:       datatypes.NewJSONType(spec),
		}).
		Error
}
