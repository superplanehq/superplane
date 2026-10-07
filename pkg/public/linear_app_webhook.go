package public

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

// HandleLinearAppWebhook receives events from a Linear OAuth application webhook.
// One URL serves every workspace that authorized the application. SuperPlane
// verifies the signing secret, then runs the triggers that listen to that team.
func (s *Server) HandleLinearAppWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxEventSize)
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	event, err := linear.ParseAppEvent(r.Header, body)
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	integrations, err := models.ListLinearIntegrationsForAppWebhook(
		database.DB(r.Context()),
		event.OrganizationID,
		event.WorkspaceKey,
	)
	if err != nil {
		log.WithError(err).Error("failed to list Linear integrations for an application webhook")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	matched := s.linearIntegrationsForSignature(r, integrations, body)
	if len(matched) == 0 {
		if len(integrations) == 0 && linear.SignatureMatches(r.Header.Get(linear.SignatureHeader), body, []byte(linear.HostedWebhookSecret())) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if len(integrations) == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		// A connection that still creates webhooks through the API has no
		// signing secret. Accept the call so Linear does not disable the
		// application webhook, and do not run any trigger.
		if !s.linearIntegrationHasWebhookSecret(r, integrations) {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "invalid signature", http.StatusForbidden)
		return
	}

	status := http.StatusOK
	for i := range matched {
		code := s.deliverLinearAppWebhook(r, &matched[i], event, body)
		if code >= http.StatusInternalServerError {
			status = code
			continue
		}
		if code >= http.StatusBadRequest && code != http.StatusNotFound && status < http.StatusBadRequest {
			status = code
		}
	}

	if status >= http.StatusBadRequest {
		http.Error(w, http.StatusText(status), status)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) linearIntegrationsForSignature(r *http.Request, integrations []models.Integration, body []byte) []models.Integration {
	signature := r.Header.Get(linear.SignatureHeader)
	matched := make([]models.Integration, 0, len(integrations))
	for i := range integrations {
		secret := s.linearWebhookSecret(r, &integrations[i])
		if !linear.SignatureMatches(signature, body, []byte(secret)) {
			continue
		}
		matched = append(matched, integrations[i])
	}
	return matched
}

func (s *Server) linearIntegrationHasWebhookSecret(r *http.Request, integrations []models.Integration) bool {
	for i := range integrations {
		if s.linearWebhookSecret(r, &integrations[i]) != "" {
			return true
		}
	}
	return false
}

func (s *Server) linearWebhookSecret(r *http.Request, integration *models.Integration) string {
	integrationContext := contexts.NewIntegrationContext(database.DB(r.Context()), nil, integration, s.encryptor, s.registry, nil)
	return linear.AppWebhookSigningSecret(integrationContext)
}

func (s *Server) deliverLinearAppWebhook(r *http.Request, integration *models.Integration, event linear.AppEvent, body []byte) int {
	webhooks, err := models.ListIntegrationWebhooks(database.DB(r.Context()), integration.ID)
	if err != nil {
		log.WithError(err).WithField("integration_id", integration.ID.String()).Error("failed to list Linear webhooks")
		return http.StatusInternalServerError
	}

	metadata := linear.Metadata{}
	_ = mapstructure.Decode(integration.Metadata.Data(), &metadata)

	status := http.StatusOK
	for i := range webhooks {
		webhook := webhooks[i]
		if webhook.State != models.WebhookStateReady || !linear.IsAppLevelWebhook(webhook.Metadata.Data()) {
			continue
		}

		config := linear.WebhookConfiguration{}
		if err := mapstructure.Decode(webhook.Configuration.Data(), &config); err != nil {
			log.WithError(err).WithField("webhook_id", webhook.ID.String()).Error("failed to read Linear webhook configuration")
			return http.StatusInternalServerError
		}
		if !linear.SubscriptionMatches(config, event, metadata.Teams) {
			continue
		}

		code := s.forwardLinearWebhook(r, webhook, body)
		if code >= http.StatusInternalServerError {
			status = code
			continue
		}
		if code >= http.StatusBadRequest && code != http.StatusNotFound && status < http.StatusBadRequest {
			status = code
		}
	}
	return status
}

func (s *Server) forwardLinearWebhook(r *http.Request, webhook models.Webhook, body []byte) int {
	request := httptest.NewRequest(http.MethodPost, r.URL.String(), bytes.NewReader(body))
	request.Header = r.Header.Clone()
	request = request.WithContext(r.Context())
	request = mux.SetURLVars(request, map[string]string{"webhookID": webhook.ID.String()})

	recorder := httptest.NewRecorder()
	s.HandleWebhook(recorder, request)
	return recorder.Code
}
