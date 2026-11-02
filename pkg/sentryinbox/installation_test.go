package sentryinbox_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	sentryintegration "github.com/superplanehq/superplane/pkg/integrations/sentry"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func installationCreatedBody(installationUUID, code string) string {
	return fmt.Sprintf(`{
		"action": "created",
		"installation": {"uuid": %q},
		"data": {"installation": {"uuid": %q, "code": %q, "organization": {"slug": "acme", "id": 1}}}
	}`, installationUUID, installationUUID, code)
}

func installationDeletedBody(installationUUID string) string {
	return fmt.Sprintf(`{"action":"deleted","installation":{"uuid":%q},"data":{"installation":{"uuid":%q}}}`, installationUUID, installationUUID)
}

func tokenResponse(token string) *http.Response {
	return jsonResponse(http.StatusCreated, fmt.Sprintf(
		`{"token":%q,"refreshToken":"refresh","expiresAt":%q}`, token, time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
	))
}

func setupInstallations(t *testing.T) *support.ResourceRegistry {
	t.Helper()
	useHostedApp(t)
	r := support.Setup(t)
	sentryintegration.EnableHostedInstallBind(r.Encryptor)
	return r
}

func storedGrant(t *testing.T, installationUUID, code string) *models.SentryAppInstallGrant {
	t.Helper()
	grant, err := models.FindSentryAppInstallGrant(database.DB(t.Context()), installationUUID, codeDigest(code), time.Now())
	require.NoError(t, err)
	return grant
}

func TestProcessorInstallationLifecycle(t *testing.T) {
	t.Run("a new installation stores exchanged tokens for the browser callback", func(t *testing.T) {
		r := setupInstallations(t)
		accepted := acceptHosted(t, "installation", installationCreatedBody("install-new", "grant-code"))
		sentryAPI := &contexts.HTTPContext{Responses: []*http.Response{tokenResponse("exchanged")}}

		result, err := newProcessor(r).WithHTTP(sentryAPI).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateProcessed, result.State)

		grant := storedGrant(t, "install-new", "grant-code")
		require.NotNil(t, grant)
		assert.NotEmpty(t, grant.AccessToken)
		require.Len(t, sentryAPI.Requests, 1)
		assert.Equal(t, "acme", grant.OrganizationSlug)
	})

	t.Run("a code that the browser callback used is recovered with the installation token grant", func(t *testing.T) {
		r := setupInstallations(t)
		accepted := acceptHosted(t, "installation", installationCreatedBody("install-used", "used-code"))
		sentryAPI := &contexts.HTTPContext{Responses: []*http.Response{
			jsonResponse(http.StatusBadRequest, `{"detail":"invalid grant"}`),
			tokenResponse("minted"),
		}}

		result, err := newProcessor(r).WithHTTP(sentryAPI).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateProcessed, result.State)
		assert.Contains(t, result.Note, "recovered")
		grant := storedGrant(t, "install-used", "used-code")
		require.NotNil(t, grant)
		assert.NotEmpty(t, grant.AccessToken)
	})

	t.Run("stored tokens are reused after a worker restart", func(t *testing.T) {
		r := setupInstallations(t)
		accepted := acceptHosted(t, "installation", installationCreatedBody("install-restart", "grant-code"))
		_, err := newProcessor(r).
			WithHTTP(&contexts.HTTPContext{Responses: []*http.Response{tokenResponse("exchanged")}}).
			Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)

		retry := acceptHosted(t, "installation", installationCreatedBody("install-restart", "grant-code"))
		noCalls := &contexts.HTTPContext{}
		result, err := newProcessor(r).WithHTTP(noCalls).Process(t.Context(), claimEntry(t, retry.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateProcessed, result.State)
		assert.Empty(t, noCalls.Requests, "Sentry accepts a code only once")
		assert.NotEmpty(t, storedGrant(t, "install-restart", "grant-code").AccessToken)
	})

	t.Run("an unrecoverable installation stays blocked with a reconnect error", func(t *testing.T) {
		r := setupInstallations(t)
		accepted := acceptHosted(t, "installation", installationCreatedBody("install-gone", "grant-code"))
		sentryAPI := &contexts.HTTPContext{Responses: []*http.Response{
			jsonResponse(http.StatusBadRequest, `{"detail":"invalid grant"}`),
			jsonResponse(http.StatusUnauthorized, `{"detail":"unauthorized"}`),
		}}

		result, err := newProcessor(r).WithHTTP(sentryAPI).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateBlocked, result.State)
		assert.Contains(t, result.Note, "reconnect required")
		assert.NotNil(t, reload(t, accepted.InboxID).PayloadCiphertext)
	})

	t.Run("a delayed creation after the deletion cannot restore the grant", func(t *testing.T) {
		r := setupInstallations(t)
		created := acceptHosted(t, "installation", installationCreatedBody("install-deleted", "grant-code"))
		deleted := acceptHosted(t, "installation", installationDeletedBody("install-deleted"))

		// The creation is retrying, so the deletion runs first.
		require.NoError(t, database.Conn().Model(&models.SentryWebhookInbox{}).
			Where("id = ?", created.InboxID).
			Update("next_attempt_at", time.Now().Add(time.Hour)).Error)
		result, err := newProcessor(r).Process(t.Context(), claimEntry(t, deleted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateProcessed, result.State)

		makeDue(t, created.InboxID)
		noCalls := &contexts.HTTPContext{}
		result, err = newProcessor(r).WithHTTP(noCalls).Process(t.Context(), claimEntry(t, created.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateFiltered, result.State)
		assert.Empty(t, noCalls.Requests)
		assert.Nil(t, storedGrant(t, "install-deleted", "grant-code"))
	})

	t.Run("a connection that the browser callback finished needs no grant", func(t *testing.T) {
		r := setupInstallations(t)
		createHostedConnection(t, r, "install-bound")
		accepted := acceptHosted(t, "installation", installationCreatedBody("install-bound", "grant-code"))
		noCalls := &contexts.HTTPContext{}

		result, err := newProcessor(r).WithHTTP(noCalls).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateProcessed, result.State)
		assert.Empty(t, noCalls.Requests)
		assert.Nil(t, storedGrant(t, "install-bound", "grant-code"))
	})

	t.Run("a worker without the public app configuration keeps the event blocked", func(t *testing.T) {
		r := setupInstallations(t)
		accepted := acceptHosted(t, "installation", installationCreatedBody("install-env", "grant-code"))
		t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "")

		result, err := newProcessor(r).Process(t.Context(), claimEntry(t, accepted.InboxID, time.Minute))
		require.NoError(t, err)
		assert.Equal(t, models.SentryWebhookStateBlocked, result.State)
	})
}
