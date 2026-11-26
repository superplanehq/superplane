package eventdistributer

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/public/ws"
)

func HandleWebhookSetupChanged(messageBody []byte, hub *ws.Hub) error {
	organizationID, err := uuid.ParseBytes(messageBody)
	if err != nil {
		return fmt.Errorf("invalid organization ID for webhook setup update: %w", err)
	}

	hub.BroadcastToWorkflow(ws.WebhookSetupTopic(organizationID), []byte(`{"event":"webhook_setup_changed"}`))
	return nil
}
