package sentry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__isSafeIntegrationSetupReturnPath(t *testing.T) {
	assert.True(t, isSafeIntegrationSetupReturnPath("/org-1/workspaces/APP/lines/line-1/setup/sentry"))
	assert.True(t, isSafeIntegrationSetupReturnPath("/onboarding"))
	assert.False(t, isSafeIntegrationSetupReturnPath("//evil.example/phishing"))
	assert.False(t, isSafeIntegrationSetupReturnPath("https://evil.example"))
	assert.False(t, isSafeIntegrationSetupReturnPath("/org-1"))
	assert.False(t, isSafeIntegrationSetupReturnPath(""))
}

func Test__afterHostedAppSetup(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")

	impl := &Sentry{}
	integrationCtx := &contexts.IntegrationContext{
		IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
		Metadata: Metadata{
			HostedApp:       true,
			State:           "csrf-state",
			StartedByUserID: "user-1",
			SetupReturnPath: "/org-1/workspaces/sp/lines/line-1/setup/sentry",
		},
	}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusCreated, `{"token":"install-token","refreshToken":"refresh-token","expiresAt":"2026-09-11T13:00:00Z"}`),
			sentryMockResponse(http.StatusOK, `{"id":"1","slug":"acme","name":"Acme"}`),
			sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"payments","name":"Payments"}]`),
			sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
		},
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/sentry/app/setup?code=oauth-code&installationId=install-1&orgSlug=acme",
		nil,
	)

	impl.afterHostedAppSetup(core.HTTPRequestContext{
		Request:         req,
		Response:        recorder,
		HTTP:            httpContext,
		Logger:          logrus.NewEntry(logrus.New()),
		Integration:     integrationCtx,
		BaseURL:         "https://app.example.com",
		WebhooksBaseURL: "https://hooks.example.com",
		OrganizationID:  "org-1",
	})

	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "https://app.example.com/org-1/workspaces/sp/lines/line-1/setup/sentry", recorder.Header().Get("Location"))
	assert.Equal(t, "ready", integrationCtx.State)

	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.True(t, metadata.HostedApp)
	assert.Equal(t, "install-1", metadata.InstallationUUID)
	require.NotNil(t, metadata.Organization)
	assert.Equal(t, "acme", metadata.Organization.Slug)
	require.Len(t, metadata.Projects, 1)

	accessToken, err := integrationCtx.Secrets().Get(SecretAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "install-token", accessToken)

	require.GreaterOrEqual(t, len(httpContext.Requests), 1)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/sentry-app-installations/install-1/authorizations/")

	var authRequest sentryAppAuthorizationRequest
	require.NoError(t, json.NewDecoder(httpContext.Requests[0].Body).Decode(&authRequest))
	assert.Equal(t, "authorization_code", authRequest.GrantType)
	assert.Equal(t, "oauth-code", authRequest.Code)
}
