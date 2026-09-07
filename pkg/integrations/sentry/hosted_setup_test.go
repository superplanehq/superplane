package sentry

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__afterHostedSetup_exchangesAndMarksReady(t *testing.T) {
	t.Setenv(config.EnvSentryAppClientID, "client-id")
	t.Setenv(config.EnvSentryAppClientSecret, "client-secret")
	t.Setenv(config.EnvSentryAppSlug, "superplane")

	integration := &contexts.IntegrationContext{
		IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
		Metadata: Metadata{
			HostedApp:       true,
			State:           "nonce-1",
			StartedByUserID: "user-1",
			SetupReturnPath: "/onboarding",
		},
	}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"token":"install-token","refreshToken":"refresh-1","expiresAt":"2026-09-08T04:00:00.000Z"}`),
			sentryMockResponse(http.StatusOK, `{"status":"installed"}`),
			sentryMockResponse(http.StatusOK, `{"id":"1","slug":"acme","name":"Acme"}`),
			sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"backend","name":"Backend"}]`),
			sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/sentry/app/setup?code=grant&installationId=install-1&orgSlug=acme",
		nil,
	)

	(&Sentry{}).HandleRequest(core.HTTPRequestContext{
		Logger:         logrus.NewEntry(logrus.New()),
		Request:        req,
		Response:       rec,
		OrganizationID: "org-1",
		BaseURL:        "https://app.example.com",
		HTTP:           httpCtx,
		Integration:    integration,
	})

	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "https://app.example.com/onboarding", rec.Header().Get("Location"))
	assert.Equal(t, "ready", integration.State)
	metadata := integration.Metadata.(Metadata)
	assert.True(t, metadata.HostedApp)
	assert.Equal(t, "install-1", metadata.InstallationID)
	assert.Empty(t, metadata.State)
	require.NotNil(t, metadata.Organization)
	assert.Equal(t, "acme", metadata.Organization.Slug)
	assert.Equal(t, "install-token", string(integration.CurrentSecrets[secretInstallationToken].Value))
	require.GreaterOrEqual(t, len(httpCtx.Requests), 2)
	assert.Contains(t, httpCtx.Requests[0].URL.String(), "/sentry-app-installations/install-1/authorizations/")
}

func Test__handleWebhook_hostedInstallationDeleted(t *testing.T) {
	t.Setenv(config.EnvSentryAppClientID, "client-id")
	t.Setenv(config.EnvSentryAppClientSecret, "client-secret")
	t.Setenv(config.EnvSentryAppSlug, "superplane")

	body := `{"action":"deleted","installation":{"uuid":"install-1"},"data":{}}`
	mac := computeWebhookSignature("client-secret", []byte(body))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sentry/app/webhook", strings.NewReader(body))
	req.Header.Set("Sentry-Hook-Signature", mac)
	req.Header.Set("Sentry-Hook-Resource", "installation")
	rec := httptest.NewRecorder()
	integration := &contexts.IntegrationContext{
		Metadata: Metadata{HostedApp: true, InstallationID: "install-1"},
		State:    "ready",
	}

	(&Sentry{}).HandleRequest(core.HTTPRequestContext{
		Logger:      logrus.NewEntry(logrus.New()),
		Request:     req,
		Response:    rec,
		Integration: integration,
	})

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "error", integration.State)
	assert.Equal(t, "Sentry uninstalled SuperPlane", integration.StateDescription)
}

func Test__Cleanup_deletesHostedInstallation(t *testing.T) {
	t.Setenv(config.EnvSentryAppClientID, "client-id")
	t.Setenv(config.EnvSentryAppClientSecret, "client-secret")
	t.Setenv(config.EnvSentryAppSlug, "superplane")

	integration := &contexts.IntegrationContext{
		Metadata: Metadata{
			HostedApp:      true,
			InstallationID: "install-1",
			Organization:   &OrganizationSummary{Slug: "acme"},
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			secretInstallationToken: {Name: secretInstallationToken, Value: []byte("install-token")},
			secretTokenExpiresAt:    {Name: secretTokenExpiresAt, Value: []byte("2099-01-01T00:00:00Z")},
		},
	}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)},
		},
	}

	err := (&Sentry{}).Cleanup(core.IntegrationCleanupContext{
		Logger:      logrus.NewEntry(logrus.New()),
		HTTP:        httpCtx,
		Integration: integration,
	})
	require.NoError(t, err)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
	assert.Equal(t, "https://sentry.io/api/0/sentry-app-installations/install-1/", httpCtx.Requests[0].URL.String())
}
