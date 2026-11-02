package public

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/sentryinbox"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

// acceptSentryWebhook is the shared durable acknowledgment of both Sentry
// webhook entry points. The caller verified the signature. The handler
// answers 200 only after the delivery commits. It never falls back to
// synchronous processing: a failed commit answers 500, and Sentry sends the
// delivery again.
func (s *Server) acceptSentryWebhook(w http.ResponseWriter, r *http.Request, delivery sentryinbox.Delivery, started time.Time) {
	accepted, err := sentryinbox.Accept(r.Context(), database.DB(r.Context()), s.encryptor, delivery)
	if err != nil {
		logging.LogSentryWebhookError("failed to queue Sentry webhook", log.Fields{
			"source":        delivery.Source,
			"hook_resource": strings.TrimSpace(delivery.Resource),
		}, err)
		captureSentryWebhookErrorToSentry(r, fmt.Errorf("queue Sentry webhook: %w", err), "source", delivery.Source)
		s.recordSentryWebhookAcceptance(r, delivery.Source, started, http.StatusInternalServerError)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	s.recordSentryWebhookAcceptance(r, delivery.Source, started, http.StatusOK)
	w.Header().Set("X-Superplane-Sentry-Delivery", accepted.InboxID.String())
	w.WriteHeader(http.StatusOK)
}

// acceptSentryIntegrationWebhook verifies and queues a webhook that Sentry
// sent to one connection. It uses the secret of that connection.
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
			"error":          err.Error(),
		})
		http.Error(w, http.StatusText(status), status)
		return
	}

	s.acceptSentryWebhook(w, r, sentryinbox.Delivery{
		Source:        models.SentryWebhookSourceIntegration,
		IntegrationID: integration.ID,
		Resource:      r.Header.Get("Sentry-Hook-Resource"),
		Body:          body,
	}, started)
}

func (s *Server) recordSentryWebhookAcceptance(r *http.Request, source string, started time.Time, status int) {
	telemetry.RecordSentryWebhookAcceptance(r.Context(), time.Since(started), source, status)
}

func isDurableSentryWebhook(r *http.Request, integration *models.Integration) bool {
	if integration == nil || integration.AppName != "sentry" || r.Method != http.MethodPost {
		return false
	}
	return config.SentryWebhookDurableAcceptance() && sentryintegration.IsWebhookPath(r.URL.Path)
}
