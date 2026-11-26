package messages

import "github.com/google/uuid"

const WebhookSetupChangedRoutingKey = "webhook-setup-changed"

func PublishWebhookSetupChanged(organizationID uuid.UUID) error {
	return Publish(CanvasExchange, WebhookSetupChangedRoutingKey, []byte(organizationID.String()))
}
