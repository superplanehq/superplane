package sentry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

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
	useFakeHostedInstallGrants(t)

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

	expiresAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusCreated, fmt.Sprintf(
				`{"token":"install-token","refreshToken":"refresh-token","expiresAt":%q}`,
				expiresAt,
			)),
			sentryMockResponse(http.StatusOK, `{"id":"1","slug":"acme","name":"Acme"}`),
			sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"payments","name":"Payments"}]`),
			sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
		},
	}

	recorder := httptest.NewRecorder()
	req := hostedSetupRequest(
		"/api/v1/sentry/app/setup?code=oauth-code&installationId=install-1&orgSlug=acme",
		"csrf-state",
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
	assert.Equal(t, expiresAt, metadata.TokenExpiresAt)
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

func Test__afterHostedAppSetup_installationIdAlias(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")
	useFakeHostedInstallGrants(t)

	impl := &Sentry{}
	integrationCtx := pendingHostedIntegration()
	expiresAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	httpContext := hostedInstallHTTPContext(expiresAt, "install-token")

	recorder := httptest.NewRecorder()
	req := hostedSetupRequest(
		"/api/v1/sentry/app/setup?code=oauth-code&installation_id=install-1&org_slug=acme",
		"csrf-state",
	)

	impl.afterHostedAppSetup(hostedSetupContext(req, recorder, httpContext, integrationCtx))

	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "ready", integrationCtx.State)
	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.Equal(t, "install-1", metadata.InstallationUUID)
}

func Test__afterHostedAppSetup_missingCode_rejectsUnknownInstallation(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")
	useFakeHostedInstallGrants(t)

	impl := &Sentry{}
	integrationCtx := pendingHostedIntegration()
	httpContext := hostedInstallHTTPContext(time.Now().Add(time.Hour).UTC().Format(time.RFC3339), "jwt-token")

	recorder := httptest.NewRecorder()
	req := hostedSetupRequest(
		"/api/v1/sentry/app/setup?installationId=install-1&orgSlug=acme",
		"csrf-state",
	)

	impl.afterHostedAppSetup(hostedSetupContext(req, recorder, httpContext, integrationCtx))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.NotEqual(t, "ready", integrationCtx.State)
	assert.Empty(t, httpContext.Requests)
}

func Test__afterHostedAppSetup_codeExchangeFailed_doesNotMint(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")
	useFakeHostedInstallGrants(t)

	impl := &Sentry{}
	integrationCtx := pendingHostedIntegration()
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusBadRequest, `{"detail":"invalid grant"}`),
			sentryMockResponse(http.StatusCreated, `{"token":"jwt-token","refreshToken":"refresh-token"}`),
		},
	}

	recorder := httptest.NewRecorder()
	req := hostedSetupRequest(
		"/api/v1/sentry/app/setup?code=stale-code&installationId=install-1&orgSlug=acme",
		"csrf-state",
	)

	impl.afterHostedAppSetup(hostedSetupContext(req, recorder, httpContext, integrationCtx))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.NotEqual(t, "ready", integrationCtx.State)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "authorization_code", decodeGrantType(t, httpContext.Requests[0]))
}

func Test__afterHostedAppSetup_finishesInterruptedInstall(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")
	useFakeHostedInstallGrants(t)

	impl := &Sentry{}
	integrationCtx := pendingHostedIntegration()
	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	metadata.InstallationUUID = "install-1"
	integrationCtx.Metadata = metadata

	expiresAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusCreated, fmt.Sprintf(
				`{"token":"minted-token","refreshToken":"refresh-token","expiresAt":%q}`,
				expiresAt,
			)),
			sentryMockResponse(http.StatusOK, `[{"id":"1","slug":"acme","name":"Acme"}]`),
			sentryMockResponse(http.StatusOK, `{"id":"1","slug":"acme","name":"Acme"}`),
			sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"payments","name":"Payments"}]`),
			sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
		},
	}

	recorder := httptest.NewRecorder()
	req := hostedSetupRequest("/api/v1/sentry/app/setup", "csrf-state")

	impl.afterHostedAppSetup(hostedSetupContext(req, recorder, httpContext, integrationCtx))

	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "ready", integrationCtx.State)

	metadata, ok = integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.Equal(t, "install-1", metadata.InstallationUUID)
	require.NotNil(t, metadata.Organization)
	assert.Equal(t, "acme", metadata.Organization.Slug)

	require.GreaterOrEqual(t, len(httpContext.Requests), 1)
	assert.Equal(t, sentryAppJWTGrantType, decodeGrantType(t, httpContext.Requests[0]))
}

func Test__afterHostedAppSetup_mismatchedState_rejects(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")

	impl := &Sentry{}
	integrationCtx := pendingHostedIntegration()
	httpContext := hostedInstallHTTPContext(time.Now().Add(time.Hour).UTC().Format(time.RFC3339), "install-token")
	recorder := httptest.NewRecorder()
	req := hostedSetupRequest(
		"/api/v1/sentry/app/setup?code=oauth-code&installationId=install-1&orgSlug=acme",
		"other-state",
	)

	impl.afterHostedAppSetup(hostedSetupContext(req, recorder, httpContext, integrationCtx))

	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.NotEqual(t, "ready", integrationCtx.State)
	assert.Empty(t, httpContext.Requests)
}

func Test__afterHostedAppSetup_usesUnclaimedWebhookGrant(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")
	grants := useFakeHostedInstallGrants(t)

	require.NoError(t, grants.Remember(hostedSentryInstall{
		InstallationUUID: "install-1",
		AccessToken:      "webhook-token",
		RefreshToken:     "webhook-refresh",
		TokenExpiresAt:   time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		Organization:     &OrganizationSummary{ID: "1", Slug: "acme", Name: "Acme"},
		Code:             "grant-code",
	}))

	impl := &Sentry{}
	integrationCtx := pendingHostedIntegration()
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"id":"1","slug":"acme","name":"Acme"}`),
			sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"payments","name":"Payments"}]`),
			sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
		},
	}

	recorder := httptest.NewRecorder()
	req := hostedSetupRequest("/api/v1/sentry/app/setup?code=grant-code&installationId=install-1", "csrf-state")

	impl.afterHostedAppSetup(hostedSetupContext(req, recorder, httpContext, integrationCtx))

	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "ready", integrationCtx.State)
	accessToken, err := integrationCtx.Secrets().Get(SecretAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "webhook-token", accessToken)
	for _, request := range httpContext.Requests {
		assert.NotContains(t, request.URL.String(), "/authorizations/")
	}

	taken, err := grants.Take("install-1", "grant-code", pendingHostedIntegration().ID().String())
	require.NoError(t, err)
	assert.Nil(t, taken)
}

func Test__afterHostedAppSetup_failedOrgLoad_keepsGrant(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")
	grants := useFakeHostedInstallGrants(t)

	require.NoError(t, grants.Remember(hostedSentryInstall{
		InstallationUUID: "install-1",
		AccessToken:      "webhook-token",
		RefreshToken:     "webhook-refresh",
		TokenExpiresAt:   time.Now().Add(time.Hour).UTC().Format(time.RFC3339),
		Organization:     &OrganizationSummary{ID: "1", Slug: "acme", Name: "Acme"},
		Code:             "grant-code",
	}))

	impl := &Sentry{}
	integrationCtx := pendingHostedIntegration()
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusInternalServerError, `{"detail":"unavailable"}`),
		},
	}

	recorder := httptest.NewRecorder()
	req := hostedSetupRequest("/api/v1/sentry/app/setup?code=grant-code&installationId=install-1", "csrf-state")

	impl.afterHostedAppSetup(hostedSetupContext(req, recorder, httpContext, integrationCtx))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.NotEqual(t, "ready", integrationCtx.State)

	taken, err := grants.Take("install-1", "grant-code", pendingHostedIntegration().ID().String())
	require.NoError(t, err)
	require.NotNil(t, taken)
	assert.Equal(t, "webhook-token", taken.AccessToken)

	stolen, err := grants.Take("install-1", "grant-code", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	require.NoError(t, err)
	assert.Nil(t, stolen)
}

func Test__ParseInstallationCreatedGrant(t *testing.T) {
	body := []byte(`{
		"action": "created",
		"data": {
			"installation": {
				"status": "pending",
				"organization": {"slug": "test-org"},
				"code": "grant-code",
				"uuid": "install-uuid"
			}
		},
		"installation": {"uuid": "install-uuid"}
	}`)

	grant, ok := ParseInstallationCreatedGrant("installation", body)
	require.True(t, ok)
	assert.Equal(t, "grant-code", grant.Code)
	assert.Equal(t, "install-uuid", grant.UUID)
	assert.Equal(t, "test-org", grant.OrgSlug)

	_, ok = ParseInstallationCreatedGrant("issue", body)
	assert.False(t, ok)
}

func Test__ForgetKnownHostedInstallation_dropsUnclaimedGrant(t *testing.T) {
	grants := useFakeHostedInstallGrants(t)

	require.NoError(t, grants.Remember(hostedSentryInstall{
		InstallationUUID: "install-1",
		AccessToken:      "webhook-token",
		Code:             "grant-code",
	}))
	require.NoError(t, ForgetKnownHostedInstallation("install-1"))

	taken, err := grants.Take("install-1", "grant-code", pendingHostedIntegration().ID().String())
	require.NoError(t, err)
	assert.Nil(t, taken)
}

func Test__ParseInstallationDeletedUUID(t *testing.T) {
	uuid, ok := ParseInstallationDeletedUUID("installation", []byte(`{
		"action": "deleted",
		"installation": {"uuid": "install-uuid"}
	}`))
	require.True(t, ok)
	assert.Equal(t, "install-uuid", uuid)

	_, ok = ParseInstallationDeletedUUID("installation", []byte(`{"action":"created","installation":{"uuid":"install-uuid"}}`))
	assert.False(t, ok)
}

func Test__redirectHostedAppInstall_doesNotBindUnclaimedGrant(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")
	grants := useFakeHostedInstallGrants(t)

	require.NoError(t, grants.Remember(hostedSentryInstall{
		InstallationUUID: "install-1",
		AccessToken:      "install-token",
		Organization:     &OrganizationSummary{Slug: "acme"},
		Code:             "grant-code",
	}))

	impl := &Sentry{}
	integrationCtx := pendingHostedIntegration()
	httpContext := &contexts.HTTPContext{}
	recorder := httptest.NewRecorder()
	req := hostedSetupRequest("/api/v1/sentry/app/install?state=csrf-state", "csrf-state")

	impl.redirectHostedAppInstall(hostedSetupContext(req, recorder, httpContext, integrationCtx))

	assert.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, HostedAppExternalInstallURL("superplane"), recorder.Header().Get("Location"))
	assert.NotEqual(t, "ready", integrationCtx.State)
}

func Test__RememberHostedInstallGrant_storesTokensForLaterSetup(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")
	grants := useFakeHostedInstallGrants(t)

	expiresAt := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusCreated, fmt.Sprintf(
				`{"token":"webhook-token","refreshToken":"webhook-refresh","expiresAt":%q}`,
				expiresAt,
			)),
		},
	}
	app, ok := HostedAppFromEnv()
	require.True(t, ok)

	require.NoError(t, RememberHostedInstallGrant(httpContext, app, InstallationGrant{
		Code:    "grant-code",
		UUID:    "install-1",
		OrgSlug: "acme",
	}))

	unclaimed, err := grants.Take("install-1", "grant-code", pendingHostedIntegration().ID().String())
	require.NoError(t, err)
	require.NotNil(t, unclaimed)
	assert.Equal(t, "webhook-token", unclaimed.AccessToken)
	assert.Equal(t, "acme", unclaimed.Organization.Slug)
}

func Test__afterHostedAppSetup_missingCode_bindsExistingSuperPlaneInstall(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")

	originalFinder := findReadyHostedSentryInstall
	findReadyHostedSentryInstall = func(organizationID, excludeID string) (*hostedSentryInstall, error) {
		return &hostedSentryInstall{
			InstallationUUID: "install-uuid",
			AccessToken:      "access-token",
			Organization:     &OrganizationSummary{ID: "1", Slug: "acme", Name: "Acme"},
		}, nil
	}
	t.Cleanup(func() {
		findReadyHostedSentryInstall = originalFinder
	})

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

	recorder := httptest.NewRecorder()
	req := hostedSetupRequest("/api/v1/sentry/app/setup", "csrf-state")

	impl.afterHostedAppSetup(core.HTTPRequestContext{
		Request:         req,
		Response:        recorder,
		Logger:          logrus.NewEntry(logrus.New()),
		Integration:     integrationCtx,
		BaseURL:         "https://app.example.com",
		WebhooksBaseURL: "https://hooks.example.com",
		OrganizationID:  "org-1",
	})

	require.Equal(t, http.StatusSeeOther, recorder.Code)
	assert.Equal(t, "ready", integrationCtx.State)
	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.Equal(t, "install-uuid", metadata.InstallationUUID)
}

type countingPersister struct {
	*contexts.IntegrationContext
	count int
}

func (c *countingPersister) Persist() error {
	c.count++
	return nil
}

func Test__hostedAccessToken_persistsRefreshedExpiry(t *testing.T) {
	t.Setenv("SUPERPLANE_SENTRY_APP_SLUG", "superplane")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_ID", "cid")
	t.Setenv("SUPERPLANE_SENTRY_APP_CLIENT_SECRET", "csecret")

	refreshedExpiry := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	integration := &countingPersister{
		IntegrationContext: &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Metadata: Metadata{
				HostedApp:        true,
				InstallationUUID: "install-1",
				TokenExpiresAt:   time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
				Organization:     &OrganizationSummary{ID: "1", Slug: "acme", Name: "Acme"},
			},
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretAccessToken:  {Name: SecretAccessToken, Value: []byte("expired-token")},
				SecretRefreshToken: {Name: SecretRefreshToken, Value: []byte("refresh-token")},
			},
		},
	}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, fmt.Sprintf(
				`{"token":"next-token","refreshToken":"next-refresh","expiresAt":%q}`,
				refreshedExpiry,
			)),
		},
	}

	client, err := NewClient(httpContext, integration)
	require.NoError(t, err)
	assert.Equal(t, "next-token", client.userToken)
	assert.Equal(t, 1, integration.count)

	metadata, ok := integration.Metadata.(Metadata)
	require.True(t, ok)
	assert.Equal(t, refreshedExpiry, metadata.TokenExpiresAt)

	accessToken, err := integration.Secrets().Get(SecretAccessToken)
	require.NoError(t, err)
	assert.Equal(t, "next-token", accessToken)
}

func pendingHostedIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
		Metadata: Metadata{
			HostedApp:       true,
			State:           "csrf-state",
			StartedByUserID: "user-1",
			SetupReturnPath: "/org-1/workspaces/sp/lines/line-1/setup/sentry",
		},
	}
}

func hostedInstallHTTPContext(expiresAt, token string) *contexts.HTTPContext {
	return &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusCreated, fmt.Sprintf(
				`{"token":%q,"refreshToken":"refresh-token","expiresAt":%q}`,
				token,
				expiresAt,
			)),
			sentryMockResponse(http.StatusOK, `{"id":"1","slug":"acme","name":"Acme"}`),
			sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"payments","name":"Payments"}]`),
			sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
		},
	}
}

func hostedSetupRequest(target, state string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.AddCookie(&http.Cookie{
		Name:  sentryAppSetupStateCookie,
		Value: url.QueryEscape(state),
	})
	return req
}

func hostedSetupContext(
	req *http.Request,
	recorder *httptest.ResponseRecorder,
	httpContext *contexts.HTTPContext,
	integration *contexts.IntegrationContext,
) core.HTTPRequestContext {
	return core.HTTPRequestContext{
		Request:         req,
		Response:        recorder,
		HTTP:            httpContext,
		Logger:          logrus.NewEntry(logrus.New()),
		Integration:     integration,
		BaseURL:         "https://app.example.com",
		WebhooksBaseURL: "https://hooks.example.com",
		OrganizationID:  "org-1",
	}
}

func decodeGrantType(t *testing.T, request *http.Request) string {
	t.Helper()
	var authRequest sentryAppAuthorizationRequest
	require.NoError(t, json.NewDecoder(request.Body).Decode(&authRequest))
	return authRequest.GrantType
}
