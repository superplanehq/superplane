package public

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
)

type linearWebhookTrackKey struct{}

type linearWebhookTrack struct {
	id          uuid.UUID
	emitted     int
	integration *models.Integration
}

func (s *Server) openLinearWebhookReceipt(ctx context.Context, webhook *models.Webhook, headers http.Header, body []byte) *linearWebhookTrack {
	if webhook == nil || webhook.AppInstallationID == nil {
		return nil
	}

	integration, err := models.FindUnscopedIntegrationInTransaction(database.DB(ctx), *webhook.AppInstallationID)
	if err != nil || integration.AppName != "linear" {
		return nil
	}

	summary := linear.SummarizeWebhook(headers, body)
	receiptID, err := models.CreateLinearWebhookReceipt(database.DB(ctx), models.LinearWebhookReceipt{
		IntegrationID:   integration.ID,
		OrganizationID:  integration.OrganizationID,
		WebhookID:       webhook.ID,
		EventType:       summary.EventType,
		Action:          summary.Action,
		IssueIdentifier: summary.IssueIdentifier,
		IssueID:         summary.IssueID,
		TeamKey:         summary.TeamKey,
		WorkspaceKey:    summary.WorkspaceKey,
		Outcome:         models.LinearWebhookOutcomePending,
		DeliveryKey:     linear.DeliveryKey(body),
	})
	if err != nil {
		logging.ForIntegration(*integration).WithError(err).Error("failed to store Linear webhook receipt")
		return nil
	}

	return &linearWebhookTrack{id: receiptID, integration: integration}
}

func withLinearWebhookTrack(ctx context.Context, track *linearWebhookTrack) context.Context {
	if track == nil {
		return ctx
	}
	return context.WithValue(ctx, linearWebhookTrackKey{}, track)
}

func linearWebhookTrackFrom(ctx context.Context) *linearWebhookTrack {
	if ctx == nil {
		return nil
	}
	track, _ := ctx.Value(linearWebhookTrackKey{}).(*linearWebhookTrack)
	return track
}

func (track *linearWebhookTrack) complete(ctx context.Context, status, subscriptions int) {
	if track == nil || track.id == uuid.Nil {
		return
	}
	err := models.UpdateLinearWebhookReceiptResult(
		database.DB(ctx),
		track.id,
		status,
		linearWebhookOutcome(status, track.emitted),
		subscriptions,
	)
	if err != nil && track.integration != nil {
		logging.ForIntegration(*track.integration).WithError(err).Error("failed to update Linear webhook receipt")
	}
}

func linearWebhookOutcome(status, emitted int) string {
	switch {
	case status == http.StatusForbidden:
		return models.LinearWebhookOutcomeRejected
	case status == http.StatusNotFound:
		return models.LinearWebhookOutcomeNoSubscription
	case status >= http.StatusOK && status < http.StatusMultipleChoices:
		if emitted == 0 {
			return models.LinearWebhookOutcomeIgnored
		}
		return models.LinearWebhookOutcomeAccepted
	default:
		return models.LinearWebhookOutcomeFailed
	}
}

type linearReceiptEvents struct {
	inner core.EventContext
	track *linearWebhookTrack
}

func (e *linearReceiptEvents) Emit(payloadType string, payload any) error {
	if payloadMap, ok := payload.(map[string]any); ok && e.track != nil && e.track.id != uuid.Nil {
		payloadMap[linear.ReceiptField] = e.track.id.String()
	}
	err := e.inner.Emit(payloadType, payload)
	if err != nil {
		return err
	}
	if e.track != nil {
		e.track.emitted++
	}
	return nil
}
