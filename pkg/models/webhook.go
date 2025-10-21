package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

const (
	WebhookHandlerTypeTrigger   = "trigger"
	WebhookHandlerTypeComponent = "component"
)

type Webhook struct {
	ID     uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	Secret []byte
}

type WebhookHandler struct {
	WebhookID uuid.UUID
	Type      string
	Spec      datatypes.JSONType[WebhookHandlerSpec]
}

type WebhookHandlerSpec struct {
	Trigger   *WebhookTriggerHandler   `json:"trigger,omitempty"`
	Component *WebhookComponentHandler `json:"component,omitempty"`
}

type WebhookTriggerHandler struct {
	WorkflowID string `json:"workflow_id"`
	NodeID     string `json:"node_id"`
}

type WebhookComponentHandler struct {
	NodeID      string `json:"node_id"`
	ExecutionID string `json:"execution_id"`
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

func (w *Webhook) Handlers() ([]WebhookHandler, error) {
	var handlers []WebhookHandler
	err := database.Conn().
		Where("webhook_id = ?", w.ID).
		Find(&handlers).
		Error

	if err != nil {
		return nil, err
	}

	return handlers, nil
}

func CreateWebhookHandler(webhookID uuid.UUID, handlerType string, spec WebhookHandlerSpec) error {
	return database.Conn().
		Create(&WebhookHandler{
			WebhookID: webhookID,
			Type:      handlerType,
			Spec:      datatypes.NewJSONType(spec),
		}).
		Error
}
