package public

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

// HandleSentryAppStart sets the setup cookie and sends the user to the
// public Sentry app external-install page.
func (s *Server) HandleSentryAppStart(w http.ResponseWriter, r *http.Request) {
	app, ok := sentry.HostedAppFromEnv()
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	integrationID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("integration")))
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	integration, err := models.FindUnscopedIntegrationInTransaction(database.DB(r.Context()), integrationID)
	if err != nil || integration.AppName != "sentry" || !isHostedSentryApp(integration) {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	if status := authorizeHostedSentryAppCallback(r.Context(), integration); status != 0 {
		writeHostedGitHubAppAuthError(w, status)
		return
	}

	metadata := sentryMetadata(integration)
	if metadata.State == "" {
		http.Error(w, "missing setup state", http.StatusBadRequest)
		return
	}

	value, err := sentry.SignSetupCookie(app.ClientSecret, integration.ID.String(), metadata.State, time.Now())
	if err != nil {
		log.WithError(err).Error("failed to sign Sentry setup cookie")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sentry.SetupCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   int(15 * time.Minute / time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, sentry.ExternalInstallURL(app.BaseURL, app.Slug), http.StatusFound)
}

// HandleSentryAppSetup finishes a public SuperPlane Sentry app install.
func (s *Server) HandleSentryAppSetup(w http.ResponseWriter, r *http.Request) {
	app, ok := sentry.HostedAppFromEnv()
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	cookie, err := r.Cookie(sentry.SetupCookieName)
	if err != nil || cookie == nil {
		http.Error(w, "missing setup cookie", http.StatusBadRequest)
		return
	}

	integrationID, nonce, err := sentry.ParseSetupCookie(app.ClientSecret, cookie.Value, time.Now())
	if err != nil {
		http.Error(w, "invalid setup cookie", http.StatusBadRequest)
		return
	}

	id, err := uuid.Parse(integrationID)
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	integration, err := models.FindUnscopedIntegrationInTransaction(database.DB(r.Context()), id)
	if err != nil || integration.AppName != "sentry" || !isHostedSentryApp(integration) {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	metadata := sentryMetadata(integration)
	if metadata.State == "" || metadata.State != nonce {
		http.Error(w, "invalid setup state", http.StatusForbidden)
		return
	}

	if status := authorizeHostedSentryAppCallback(r.Context(), integration); status != 0 {
		writeHostedGitHubAppAuthError(w, status)
		return
	}

	s.dispatchIntegrationRequest(w, r, integration)
}

// HandleSentryAppWebhook receives events for the public SuperPlane Sentry app.
func (s *Server) HandleSentryAppWebhook(w http.ResponseWriter, r *http.Request) {
	app, ok := sentry.HostedAppFromEnv()
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	if err := sentry.VerifyWebhookSignature(r.Header.Get("Sentry-Hook-Signature"), body, []byte(app.ClientSecret)); err != nil {
		http.Error(w, "invalid webhook payload", http.StatusForbidden)
		return
	}

	installationID := sentryInstallationID(body)
	if installationID == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	integrations, err := models.ListSentryIntegrationsByInstallationID(database.DB(r.Context()), installationID)
	if err != nil {
		log.WithError(err).Error("failed to list Sentry app integrations")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	for i := range integrations {
		cloned, err := cloneRequestWithBody(r, body)
		if err != nil {
			log.WithError(err).Error("failed to clone Sentry app webhook request")
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		s.dispatchIntegrationRequest(httptest.NewRecorder(), cloned, &integrations[i])
	}

	w.WriteHeader(http.StatusOK)
}

func authorizeHostedSentryAppCallback(ctx context.Context, integration *models.Integration) int {
	account, ok := middleware.GetEffectiveAccountFromContext(ctx)
	if !ok {
		return http.StatusUnauthorized
	}

	user, err := models.FindActiveHumanUserByAccountAndOrganization(
		database.DB(ctx),
		integration.OrganizationID,
		account.ID,
	)
	if err != nil {
		return http.StatusForbidden
	}

	metadata := sentryMetadata(integration)
	if !metadata.AllowsStartedBy(user.ID.String()) {
		return http.StatusForbidden
	}

	return 0
}

func isHostedSentryApp(integration *models.Integration) bool {
	if integration == nil {
		return false
	}

	return sentryMetadata(integration).HostedApp
}

func sentryMetadata(integration *models.Integration) sentry.Metadata {
	var metadata sentry.Metadata
	if integration == nil {
		return metadata
	}
	_ = mapstructure.Decode(integration.Metadata.Data(), &metadata)
	return metadata
}

func sentryInstallationID(body []byte) string {
	var payload struct {
		Installation struct {
			UUID string `json:"uuid"`
		} `json:"installation"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Installation.UUID)
}
