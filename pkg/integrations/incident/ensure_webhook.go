package incident

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// EnsureWebhookExists creates one webhook for the incident integration if none exist.
// This allows the integration page to show the webhook URL immediately so users can add it in incident.io.
// The webhook is created with no secret so HandleWebhook will fall back to the integration-level
// webhookSigningSecret until the user completes setup and the secret is persisted via SetSecret.
func EnsureWebhookExists(tx *gorm.DB, integrationID uuid.UUID) error {
	webhooks, err := models.ListIntegrationWebhooks(tx, integrationID)
	if err != nil || len(webhooks) > 0 {
		return err
	}

	webhookID := uuid.New()
	now := time.Now()
	config := WebhookConfiguration{
		Events:            []string{EventIncidentCreatedV2, EventIncidentUpdatedV2},
		SigningSecretHash: "",
	}

	webhook := models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStatePending,
		Secret:            nil,
		Configuration:     datatypes.NewJSONType(any(config)),
		AppInstallationID: &integrationID,
		CreatedAt:         &now,
	}

	return tx.Create(&webhook).Error
}
