package incident

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// EnsureWebhookExists creates one webhook for the incident integration if none exist.
// This allows the integration page to show the webhook URL immediately so users can add it in incident.io.
func EnsureWebhookExists(tx *gorm.DB, integrationID uuid.UUID, encryptor crypto.Encryptor) error {
	webhooks, err := models.ListIntegrationWebhooks(tx, integrationID)
	if err != nil || len(webhooks) > 0 {
		return err
	}

	webhookID := uuid.New()
	_, encryptedKey, err := crypto.NewRandomKey(context.Background(), encryptor, webhookID.String())
	if err != nil {
		return err
	}

	now := time.Now()
	config := WebhookConfiguration{
		Events:        []string{EventIncidentCreatedV2, EventIncidentUpdatedV2},
		SigningSecret: "",
	}

	webhook := models.Webhook{
		ID:                webhookID,
		State:             models.WebhookStatePending,
		Secret:            encryptedKey,
		Configuration:     datatypes.NewJSONType(any(config)),
		AppInstallationID: &integrationID,
		CreatedAt:         &now,
	}

	return tx.Create(&webhook).Error
}
