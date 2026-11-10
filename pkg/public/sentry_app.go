package public

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

// HandleSentryAppInstall starts a public SuperPlane Sentry app install. The
// CSRF state finds the pending SuperPlane connection, then the browser opens
// Sentry.
func (s *Server) HandleSentryAppInstall(w http.ResponseWriter, r *http.Request) {
	if rejectInSupportSession(w, r) {
		return
	}

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

	if _, ok := sentryintegration.HostedAppFromEnv(); !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	sentryintegration.EnableHostedInstallBind(s.encryptor)
	setSentryAppSetupStateCookie(w, state)
	s.dispatchIntegrationRequest(w, r, integration)
}

// HandleSentryAppSetup finishes a public SuperPlane Sentry app install.
// Sentry sends every install to this one Redirect URL. The CSRF cookie or
// the signed-in user finds the pending SuperPlane connection.
func (s *Server) HandleSentryAppSetup(w http.ResponseWriter, r *http.Request) {
	if rejectInSupportSession(w, r) {
		return
	}

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

// HandleSentryAppWebhook receives events for the public SuperPlane Sentry
// app. One Sentry install can map to more than one SuperPlane connection.
// It verifies the signature with the app secret and commits matching issue
// events before it answers.
func (s *Server) HandleSentryAppWebhook(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	app, ok := sentryintegration.HostedAppFromEnv()
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	if err := sentryintegration.VerifyWebhookSignature(r.Header.Get("Sentry-Hook-Signature"), body, []byte(app.ClientSecret)); err != nil {
		logging.LogSentryWebhookWarn("Sentry app webhook was rejected", log.Fields{
			"hook_resource": strings.TrimSpace(r.Header.Get("Sentry-Hook-Resource")),
			"status":        http.StatusBadRequest,
		})
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	summary := sentryintegration.SummarizeWebhook(r.Header.Get("Sentry-Hook-Resource"), body)
	if installationUUID, deleted := sentryintegration.ParseInstallationDeletedUUID(r.Header.Get("Sentry-Hook-Resource"), body); deleted {
		if err := sentryintegration.ForgetKnownHostedInstallation(installationUUID); err != nil {
			logging.LogSentryWebhookError("failed to drop the grant of a deleted Sentry app install", log.Fields{
				"installation_uuid": installationUUID,
			}, err)
		}
	}

	integrations, err := models.ListSentryIntegrationsByInstallationUUID(database.DB(r.Context()), summary.InstallationUUID)
	if err != nil {
		logging.LogSentryWebhookError("failed to list Sentry app integrations", log.Fields{
			"installation_uuid": summary.InstallationUUID,
		}, err)
		captureSentryWebhookErrorToSentry(r, err, "installation_uuid", summary.InstallationUUID)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if len(integrations) == 0 {
		if err := rememberPendingHostedSentryInstall(r, body); err != nil {
			captureSentryWebhookErrorToSentry(r, err, "installation_uuid", summary.InstallationUUID)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
	s.acceptSentryWebhook(w, r, body, integrations, started)
}

// rememberPendingHostedSentryInstall stores the grant from an unclaimed
// installation.created event for the setup callback. It does not call
// Sentry, so a failed write keeps the code unused and Sentry can send the
// event again.
func rememberPendingHostedSentryInstall(r *http.Request, body []byte) error {
	grant, ok := sentryintegration.ParseInstallationCreatedGrant(r.Header.Get("Sentry-Hook-Resource"), body)
	if !ok || grant.Code == "" {
		return nil
	}

	err := sentryintegration.RememberHostedInstallGrant(grant)
	if err == nil {
		return nil
	}
	logging.LogSentryWebhookError("failed to store unclaimed Sentry app install", log.Fields{
		"installation_uuid":        grant.UUID,
		"sentry_organization_slug": grant.OrgSlug,
		"sentry_organization_id":   grant.OrgID,
	}, err)
	return fmt.Errorf("store Sentry install grant for installation %s: %w", grant.UUID, err)
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

	var metadata sentryintegration.Metadata
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

	var metadata sentryintegration.Metadata
	if err := mapstructure.Decode(integration.Metadata.Data(), &metadata); err != nil {
		return false
	}
	return metadata.HostedApp
}

func setSentryAppSetupStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "sentry_app_setup_state",
		Value:    state,
		Path:     "/",
		MaxAge:   1200,
		HttpOnly: true,
		Secure:   true,
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

func captureSentryWebhookErrorToSentry(r *http.Request, err error, tags ...string) {
	hub := sentry.CurrentHub()
	if hub == nil || hub.Client() == nil {
		return
	}
	hub.WithScope(func(scope *sentry.Scope) {
		applySentryWebhookErrorTags(scope, r, tags)
		hub.CaptureException(err)
	})
}

func applySentryWebhookErrorTags(scope *sentry.Scope, r *http.Request, tags []string) {
	if len(tags)%2 != 0 {
		return
	}
	for i := 0; i < len(tags); i += 2 {
		key := tags[i]
		value := tags[i+1]
		if key == "" || value == "" {
			continue
		}
		scope.SetTag(key, value)
	}
	if r != nil {
		if resource := r.Header.Get("Sentry-Hook-Resource"); resource != "" {
			scope.SetTag("hook_resource", resource)
		}
		scope.SetRequest(r)
	}
}
