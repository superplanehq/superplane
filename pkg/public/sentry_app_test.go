package public

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"gorm.io/datatypes"
)

func Test__HandleSentryAppStart_setsCookieAndRedirects(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()
	t.Setenv(config.EnvSentryAppClientID, "client-id")
	t.Setenv(config.EnvSentryAppClientSecret, "client-secret")
	t.Setenv(config.EnvSentryAppSlug, "superplane")

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "sentry", "sentry-hosted", nil)
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"state":           "nonce-1",
		"hostedApp":       true,
		"startedByUserID": r.User.String(),
	})
	require.NoError(t, database.Conn().Save(integration).Error)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sentry/app/start?integration="+integration.ID.String(), nil)
	req = req.WithContext(accountContext(r.Account))
	rec := httptest.NewRecorder()

	(&Server{}).HandleSentryAppStart(rec, req)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, "https://sentry.io/sentry-apps/superplane/external-install/", rec.Header().Get("Location"))
	var setupCookie *http.Cookie
	for _, item := range rec.Result().Cookies() {
		if item.Name == sentry.SetupCookieName {
			setupCookie = item
		}
	}
	require.NotNil(t, setupCookie)
	integrationID, nonce, err := sentry.ParseSetupCookie("client-secret", setupCookie.Value, time.Now())
	require.NoError(t, err)
	assert.Equal(t, integration.ID.String(), integrationID)
	assert.Equal(t, "nonce-1", nonce)
}

func Test__HandleSentryAppSetup_rejectsMissingCookie(t *testing.T) {
	t.Setenv(config.EnvSentryAppClientID, "client-id")
	t.Setenv(config.EnvSentryAppClientSecret, "client-secret")
	t.Setenv(config.EnvSentryAppSlug, "superplane")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sentry/app/setup?code=x&installationId=y", nil)
	rec := httptest.NewRecorder()
	(&Server{}).HandleSentryAppSetup(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func Test__HandleSentryAppWebhook_notConfigured(t *testing.T) {
	t.Setenv(config.EnvSentryAppClientID, "")
	t.Setenv(config.EnvSentryAppClientSecret, "")
	t.Setenv(config.EnvSentryAppSlug, "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sentry/app/webhook", nil)
	rec := httptest.NewRecorder()
	(&Server{}).HandleSentryAppWebhook(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func Test__authorizeHostedSentryAppCallback(t *testing.T) {
	r := support.Setup(t)
	defer r.Close()

	integration, err := models.CreateIntegration(uuid.New(), r.Organization.ID, "sentry", "sentry-hosted", nil)
	require.NoError(t, err)
	integration.Metadata = datatypes.NewJSONType(map[string]any{
		"hostedApp":       true,
		"startedByUserID": r.User.String(),
	})
	require.NoError(t, database.Conn().Save(integration).Error)

	assert.Equal(t, http.StatusUnauthorized, authorizeHostedSentryAppCallback(t.Context(), integration))
	assert.Equal(t, 0, authorizeHostedSentryAppCallback(accountContext(r.Account), integration))
}

func Test__sentryInstallationID(t *testing.T) {
	assert.Equal(t, "install-1", sentryInstallationID([]byte(`{"installation":{"uuid":"install-1"}}`)))
	assert.Empty(t, sentryInstallationID([]byte(`{}`)))
}
