// Package sentryinbox stores verified Sentry webhooks durably and processes
// them later. The API answers Sentry after the delivery commits to
// PostgreSQL. A worker then runs enrichment, installation grants, and trigger
// delivery, and retries unfinished work until it is done.
package sentryinbox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/encryption"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

// Delivery is one verified webhook request. The caller verifies the
// signature before it calls Accept.
type Delivery struct {
	Source string
	// IntegrationID routes a delivery that Sentry sent to one connection.
	// The public app routes by the installation in the payload instead.
	IntegrationID uuid.UUID
	Resource      string
	Body          []byte
}

// Accepted describes a committed delivery.
type Accepted struct {
	InboxID      uuid.UUID
	ReceiptID    uuid.UUID
	Integrations int
	Targets      int
}

// Accept commits the receipt, the encrypted inbox entry, and the known
// trigger targets in one transaction. The caller answers Sentry with 200
// only after Accept returns no error. An encryption or database failure
// returns an error, and the caller must answer with a server error so that
// Sentry sends the delivery again.
func Accept(ctx context.Context, db *gorm.DB, encryptor crypto.Encryptor, delivery Delivery) (Accepted, error) {
	now := time.Now().UTC()
	summary := sentryintegration.SummarizeWebhook(delivery.Resource, delivery.Body)

	entry := models.NewSentryWebhookInbox(delivery.Source, summary.Resource, now)
	entry.InstallationUUID = summary.InstallationUUID
	entry.Action = summary.Action
	if delivery.Source == models.SentryWebhookSourceIntegration {
		integrationID := delivery.IntegrationID
		entry.IntegrationID = &integrationID
	}

	ciphertext, err := encryption.ForDB(db, encryptor).Encrypt(ctx, delivery.Body, []byte(entry.ID.String()))
	if err != nil {
		return Accepted{}, fmt.Errorf("encrypt sentry webhook: %w", err)
	}
	entry.PayloadCiphertext = ciphertext

	accepted := Accepted{InboxID: entry.ID}
	err = db.Transaction(func(tx *gorm.DB) error {
		resolution, err := resolveTargets(tx, &entry, now)
		if err != nil {
			return err
		}
		entry.TargetsResolved = resolution.resolved

		receipt := models.SentryWebhookReceipt{
			ReceivedAt:       now,
			HookResource:     summary.Resource,
			Action:           summary.Action,
			InstallationUUID: summary.InstallationUUID,
			OrganizationSlug: summary.OrganizationSlug,
			ProjectSlug:      summary.ProjectSlug,
			IssueID:          summary.IssueID,
			IssueShortID:     summary.IssueShortID,
			HTTPStatus:       200,
			Outcome:          models.SentryWebhookStateQueued,
			IntegrationCount: resolution.integrations,
		}
		if err := models.CreateSentryWebhookDelivery(tx, receipt, &entry, resolution.targets); err != nil {
			return err
		}

		accepted.Integrations = resolution.integrations
		accepted.Targets = len(resolution.targets)
		return nil
	})
	if err != nil {
		return Accepted{}, fmt.Errorf("store sentry webhook: %w", err)
	}

	if entry.ReceiptID != nil {
		accepted.ReceiptID = *entry.ReceiptID
	}
	logging.LogSentryWebhookInfo("Sentry webhook queued", log.Fields{
		"inbox_id":          accepted.InboxID.String(),
		"receipt_id":        accepted.ReceiptID.String(),
		"source":            delivery.Source,
		"hook_resource":     summary.Resource,
		"action":            summary.Action,
		"installation_uuid": summary.InstallationUUID,
		"integration_count": accepted.Integrations,
		"target_count":      accepted.Targets,
	})
	return accepted, nil
}
