package public

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

// HandleSentryAppInstall starts a public SuperPlane Sentry app install. The
// CSRF state finds the pending SuperPlane connection, then the browser opens
// Sentry.
func (s *Server) HandleSentryAppInstall(w http.ResponseWriter, r *http.Request) {
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}

	integration, err := models.FindSentryIntegrationByAppState(database.DB(r.Context()), state)
	if err != nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	if !isHostedSentryApp(integration) {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	if status := authorizeHostedSentryAppCallback(r.Context(), integration); status != 0 {
		writeHostedGitHubAppAuthError(w, status)
		return
	}

	if _, ok := sentry.HostedAppFromEnv(); !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	sentry.EnableHostedInstallBind(s.encryptor)
	setSentryAppSetupStateCookie(w, state)
	s.dispatchIntegrationRequest(w, r, integration)
}

// HandleSentryAppSetup finishes a public SuperPlane Sentry app install.
// Sentry sends every install to this one Redirect URL. The CSRF cookie or
// the signed-in user finds the pending SuperPlane connection.
func (s *Server) HandleSentryAppSetup(w http.ResponseWriter, r *http.Request) {
	integration, err := findSentryAppSetupIntegration(r)
	if err != nil || integration == nil {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	if !isHostedSentryApp(integration) {
		http.Error(w, "integration not found", http.StatusNotFound)
		return
	}

	if status := authorizeHostedSentryAppCallback(r.Context(), integration); status != 0 {
		writeHostedGitHubAppAuthError(w, status)
		return
	}

	s.dispatchIntegrationRequest(w, r, integration)
}

// HandleSentryAppWebhook receives issue events for the public SuperPlane
// Sentry app. One Sentry install can map to more than one SuperPlane
// connection.
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
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	installationUUID := sentryInstallationUUID(body)
	if uuid, ok := sentry.ParseInstallationDeletedUUID(r.Header.Get("Sentry-Hook-Resource"), body); ok {
		sentry.ForgetKnownHostedInstallation(uuid)
	}

	integrations, err := models.ListSentryIntegrationsByInstallationUUID(database.DB(r.Context()), installationUUID)
	if err != nil {
		log.WithError(err).Error("failed to list Sentry app integrations")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if len(integrations) == 0 {
		s.claimPendingHostedSentryInstall(r, app, body)
		w.WriteHeader(http.StatusOK)
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

func (s *Server) claimPendingHostedSentryInstall(r *http.Request, app sentry.HostedApp, body []byte) {
	grant, ok := sentry.ParseInstallationCreatedGrant(r.Header.Get("Sentry-Hook-Resource"), body)
	if !ok {
		return
	}

	if err := s.rememberHostedSentryGrant(app, grant); err != nil {
		log.WithError(err).Error("failed to store unclaimed Sentry app install")
	}
}

func (s *Server) rememberHostedSentryGrant(app sentry.HostedApp, grant sentry.InstallationGrant) error {
	if s.registry == nil || s.registry.HTTPContext() == nil {
		return nil
	}
	return sentry.RememberHostedInstallGrant(s.registry.HTTPContext(), app, grant)
}

func findSentryAppSetupIntegration(r *http.Request) (*models.Integration, error) {
	if state := sentryAppSetupStateFromRequest(r); state != "" {
		integration, err := models.FindSentryIntegrationByAppState(database.DB(r.Context()), state)
		if err == nil {
			return integration, nil
		}
	}

	account, ok := middleware.GetEffectiveAccountFromContext(r.Context())
	if !ok {
		return nil, http.ErrNoCookie
	}

	users, err := models.ListActiveHumanUsersForAccount(database.DB(r.Context()), account.ID)
	if err != nil {
		return nil, err
	}
	for i := range users {
		integration, err := models.FindPendingHostedSentryIntegration(database.DB(r.Context()), users[i].ID.String())
		if err == nil {
			return integration, nil
		}
	}
	return nil, http.ErrNoCookie
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

	var metadata sentry.Metadata
	if err := mapstructure.Decode(integration.Metadata.Data(), &metadata); err != nil {
		return http.StatusForbidden
	}

	if !metadata.AllowsStartedBy(user.ID.String()) {
		return http.StatusForbidden
	}

	return 0
}

func hostedSentryAppBrowserCallbackStatus(ctx context.Context, r *http.Request, integration *models.Integration) int {
	if !isHostedSentryAppBrowserCallback(r, integration) {
		return 0
	}

	return authorizeHostedSentryAppCallback(ctx, integration)
}

func isHostedSentryAppBrowserCallback(r *http.Request, integration *models.Integration) bool {
	if r == nil || integration == nil || integration.AppName != "sentry" {
		return false
	}

	path := r.URL.Path
	if !strings.HasSuffix(path, "/setup") && !strings.HasSuffix(path, "/install") {
		return false
	}

	return isHostedSentryApp(integration)
}

func isHostedSentryApp(integration *models.Integration) bool {
	if integration == nil || integration.AppName != "sentry" {
		return false
	}

	var metadata sentry.Metadata
	if err := mapstructure.Decode(integration.Metadata.Data(), &metadata); err != nil {
		return false
	}
	return metadata.HostedApp
}

func sentryInstallationUUID(body []byte) string {
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

func setSentryAppSetupStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "sentry_app_setup_state",
		Value:    state,
		Path:     "/",
		MaxAge:   1200,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func sentryAppSetupStateFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("sentry_app_setup_state")
	if err != nil || cookie == nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}
