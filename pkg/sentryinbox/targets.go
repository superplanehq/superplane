package sentryinbox

import (
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type targetResolution struct {
	// resolved is true when SuperPlane knows every target of the entry.
	// It stays false while no connection matches, so a later attempt can
	// find a repaired connection.
	resolved     bool
	integrations int
	targets      []models.SentryWebhookDeliveryTarget
}

// resolveTargets finds the trigger subscriptions that must receive an entry.
// Public app installation events need no target: the worker stores the
// installation grant instead.
func resolveTargets(tx *gorm.DB, entry *models.SentryWebhookInbox, now time.Time) (targetResolution, error) {
	if isHostedLifecycleEvent(entry) {
		return targetResolution{resolved: true}, nil
	}

	integrations, err := findConnections(tx, entry)
	if err != nil {
		return targetResolution{}, err
	}
	if len(integrations) == 0 {
		return targetResolution{}, nil
	}

	resolution := targetResolution{resolved: true, integrations: len(integrations)}
	for i := range integrations {
		subscriptions, err := models.ListIntegrationSubscriptions(tx, integrations[i].ID)
		if err != nil {
			return targetResolution{}, err
		}

		for _, subscription := range subscriptions {
			wanted, err := sentryintegration.SubscriptionWantsResource(subscription.Configuration.Data(), entry.HookResource)
			if err != nil {
				logging.LogSentryWebhookWarn("Sentry subscription configuration was skipped", log.Fields{
					"integration_id": integrations[i].ID.String(),
					"workflow_id":    subscription.WorkflowID.String(),
					"node_id":        subscription.NodeID,
					"error":          err.Error(),
				})
				continue
			}
			if !wanted {
				continue
			}

			resolution.targets = append(resolution.targets, models.NewSentryWebhookDeliveryTarget(
				entry.ID, integrations[i].ID, subscription.WorkflowID, subscription.NodeID, now,
			))
		}
	}
	return resolution, nil
}

func findConnections(tx *gorm.DB, entry *models.SentryWebhookInbox) ([]models.Integration, error) {
	if entry.Source == models.SentryWebhookSourceHostedApp {
		return models.ListSentryIntegrationsByInstallationUUID(tx, entry.InstallationUUID)
	}

	if entry.IntegrationID == nil {
		return nil, nil
	}
	integration, err := models.FindUnscopedIntegrationInTransaction(tx, *entry.IntegrationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return []models.Integration{*integration}, nil
}

func isHostedLifecycleEvent(entry *models.SentryWebhookInbox) bool {
	return entry.Source == models.SentryWebhookSourceHostedApp && entry.IsInstallationEvent()
}
