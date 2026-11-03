package public

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

// acceptSentryWebhook commits the receipt and all matching workflow events
// before answering Sentry. RabbitMQ only speeds up routing; the event router
// also polls pending events from the database.
func (s *Server) acceptSentryWebhook(w http.ResponseWriter, r *http.Request, body []byte, integrations []models.Integration, started time.Time) {
	resource := r.Header.Get("Sentry-Hook-Resource")
	message, err := sentryintegration.ParseWebhookMessage(resource, body)
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	summary := sentryintegration.SummarizeWebhook(resource, body)
	newEvents := []models.CanvasEvent{}
	var receiptID uuid.UUID
	err = database.DB(r.Context()).Transaction(func(tx *gorm.DB) error {
		outcome := models.SentryWebhookOutcomeAccepted
		if len(integrations) == 0 {
			outcome = models.SentryWebhookOutcomeNoConnection
		}
		var createErr error
		receiptID, createErr = models.CreateSentryWebhookReceipt(tx, models.SentryWebhookReceipt{
			HookResource:     summary.Resource,
			Action:           summary.Action,
			InstallationUUID: summary.InstallationUUID,
			OrganizationSlug: summary.OrganizationSlug,
			ProjectSlug:      summary.ProjectSlug,
			IssueID:          summary.IssueID,
			IssueShortID:     summary.IssueShortID,
			HTTPStatus:       http.StatusOK,
			Outcome:          outcome,
			IntegrationCount: len(integrations),
		})
		if createErr != nil {
			return fmt.Errorf("store Sentry webhook receipt: %w", createErr)
		}

		message.ReceiptID = receiptID.String()
		for i := range integrations {
			integration := &integrations[i]
			integrationCtx := contexts.NewIntegrationContext(tx, nil, integration, s.encryptor, s.registry, func(events []models.CanvasEvent) {
				newEvents = append(newEvents, events...)
			})
			if err := sentryintegration.DispatchWebhookMessage(integrationCtx, message, logging.ForIntegration(*integration)); err != nil {
				return fmt.Errorf("dispatch Sentry webhook to integration %s: %w", integration.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		logging.LogSentryWebhookError("failed to persist Sentry webhook", log.Fields{
			"hook_resource": strings.TrimSpace(resource),
		}, err)
		captureSentryWebhookErrorToSentry(r, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set(sentryintegration.HeaderWebhookReceipt, receiptID.String())
	w.WriteHeader(http.StatusOK)
	logging.LogSentryWebhookInfo("Sentry webhook accepted", log.Fields{
		"hook_resource":     message.Resource,
		"action":            message.Action,
		"integration_count": len(integrations),
		"event_count":       len(newEvents),
		"duration_ms":       time.Since(started).Milliseconds(),
	})
	go notifySentryWorkflowEvents(newEvents)
}

func notifySentryWorkflowEvents(events []models.CanvasEvent) {
	for i := range events {
		if err := messages.PublishCanvasEventCreatedMessage(&events[i]); err != nil {
			logging.LogSentryWebhookWarn("Sentry webhook event notification failed; the event router will poll it", log.Fields{
				"event_id": events[i].ID.String(),
				"error":    err.Error(),
			})
		}
	}
}

// acceptSentryIntegrationWebhook verifies a direct connection before it
// admits events from that connection.
func (s *Server) acceptSentryIntegrationWebhook(w http.ResponseWriter, r *http.Request, integration *models.Integration) {
	started := time.Now()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	integrationCtx := contexts.NewIntegrationContext(database.DB(r.Context()), nil, integration, s.encryptor, s.registry, nil)
	err = sentryintegration.AuthenticateIntegrationWebhook(integrationCtx, r.Header.Get("Sentry-Hook-Signature"), body)
	if err != nil {
		status := http.StatusForbidden
		if !errors.Is(err, sentryintegration.ErrWebhookNotAuthenticated) {
			status = http.StatusInternalServerError
		}
		logging.LogSentryWebhookWarn("Sentry webhook was rejected", log.Fields{
			"integration_id": integration.ID.String(),
			"hook_resource":  strings.TrimSpace(r.Header.Get("Sentry-Hook-Resource")),
			"status":         status,
		})
		http.Error(w, http.StatusText(status), status)
		return
	}

	s.acceptSentryWebhook(w, r, body, []models.Integration{*integration}, started)
}

func isSentryWebhookRequest(r *http.Request, integration *models.Integration) bool {
	return integration != nil && integration.AppName == "sentry" &&
		r.Method == http.MethodPost && sentryintegration.IsWebhookPath(r.URL.Path)
}
