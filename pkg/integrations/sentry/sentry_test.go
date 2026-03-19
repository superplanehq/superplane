package sentry

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func sentryMockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func Test__Sentry__Sync(t *testing.T) {
	impl := &Sentry{}

	t.Run("missing app credentials -> setup prompt", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{},
			Secrets:       map[string]core.IntegrationSecret{},
		}

		err := impl.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			Integration:     integrationCtx,
			BaseURL:         "https://app.example.com",
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Contains(t, integrationCtx.BrowserAction.Description, "/callback")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "/events")
		assert.Equal(t, "https://sentry.io/settings/account/developer-settings/", integrationCtx.BrowserAction.URL)
	})

	t.Run("configured app without tokens -> install prompt", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"appSlug":      "superplane",
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
			Secrets: map[string]core.IntegrationSecret{},
		}

		err := impl.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, "https://sentry.io/sentry-apps/superplane/external-install/", integrationCtx.BrowserAction.URL)
	})

	t.Run("connected app refreshes token and syncs metadata", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":      "https://sentry.io",
				"appSlug":      "superplane",
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
			Metadata: Metadata{
				InstallationID: "install-123",
				Organization: &OrganizationSummary{
					Slug: "example",
				},
			},
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessTokenSecret:  {Name: OAuthAccessTokenSecret, Value: []byte("old-access")},
				OAuthRefreshTokenSecret: {Name: OAuthRefreshTokenSecret, Value: []byte("refresh-token")},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `{"token":"new-access","refreshToken":"new-refresh","expiresAt":"2999-01-01T00:00:00Z"}`),
				sentryMockResponse(http.StatusOK, `{"id":"1","slug":"example","name":"Example Org"}`),
				sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"backend","name":"Backend"}]`),
				sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
			},
		}

		err := impl.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          httpContext,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		assert.Equal(t, []byte("new-access"), integrationCtx.Secrets[OAuthAccessTokenSecret].Value)
		assert.Equal(t, []byte("new-refresh"), integrationCtx.Secrets[OAuthRefreshTokenSecret].Value)
		require.Len(t, integrationCtx.ResyncRequests, 1)

		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Organization)
		assert.Equal(t, "Example Org", metadata.Organization.Name)
		require.Len(t, metadata.Projects, 1)
		assert.Equal(t, "backend", metadata.Projects[0].Slug)
		require.Len(t, metadata.Teams, 1)
		assert.Equal(t, "platform", metadata.Teams[0].Slug)
	})
}

func Test__Sentry__HandleCallback(t *testing.T) {
	impl := &Sentry{}

	integrationCtx := &contexts.IntegrationContext{
		IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
		Configuration: map[string]any{
			"baseUrl":      "https://sentry.io",
			"appSlug":      "superplane",
			"clientId":     "client-id",
			"clientSecret": "client-secret",
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"token":"new-access","refreshToken":"new-refresh","expiresAt":"2999-01-01T00:00:00Z"}`),
			sentryMockResponse(http.StatusOK, `{"app":{"uuid":"app-1","slug":"superplane"},"organization":{"slug":"example"},"uuid":"install-123"}`),
			sentryMockResponse(http.StatusOK, `{"id":"1","slug":"example","name":"Example Org"}`),
			sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"backend","name":"Backend"}]`),
			sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
		},
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/integrations/test/callback?code=test-code&installationId=install-123", nil)
	response := httptest.NewRecorder()

	impl.HandleRequest(core.HTTPRequestContext{
		Logger:         logrus.NewEntry(logrus.New()),
		Request:        request,
		Response:       response,
		BaseURL:        "https://app.example.com",
		OrganizationID: "org-123",
		HTTP:           httpContext,
		Integration:    integrationCtx,
	})

	require.Equal(t, http.StatusSeeOther, response.Code)
	assert.Equal(t, []byte("new-access"), integrationCtx.Secrets[OAuthAccessTokenSecret].Value)
	assert.Equal(t, []byte("new-refresh"), integrationCtx.Secrets[OAuthRefreshTokenSecret].Value)
	assert.Equal(t, "ready", integrationCtx.State)
	require.Len(t, integrationCtx.ResyncRequests, 1)

	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.Equal(t, "install-123", metadata.InstallationID)
	require.NotNil(t, metadata.Organization)
	assert.Equal(t, "example", metadata.Organization.Slug)
	assert.Equal(t, "Example Org", metadata.Organization.Name)
}

func Test__Sentry__HandleInstallationDeleted(t *testing.T) {
	impl := &Sentry{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl":      "https://sentry.io",
			"appSlug":      "superplane",
			"clientId":     "client-id",
			"clientSecret": "client-secret",
		},
		Metadata: Metadata{
			InstallationID: "install-123",
			Organization: &OrganizationSummary{
				Slug: "example",
			},
			Projects: []ProjectSummary{{Slug: "backend", Name: "Backend"}},
		},
		Secrets: map[string]core.IntegrationSecret{
			OAuthAccessTokenSecret:  {Name: OAuthAccessTokenSecret, Value: []byte("access")},
			OAuthRefreshTokenSecret: {Name: OAuthRefreshTokenSecret, Value: []byte("refresh")},
		},
	}

	body := []byte(`{"action":"deleted","installation":{"uuid":"install-123"},"data":{"installation":{"uuid":"install-123"}}}`)
	signature := computeWebhookSignature("client-secret", body)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/test/events", bytes.NewReader(body))
	request.Header.Set("Sentry-Hook-Resource", "installation")
	request.Header.Set("Sentry-Hook-Signature", signature)
	response := httptest.NewRecorder()

	impl.HandleRequest(core.HTTPRequestContext{
		Logger:      logrus.NewEntry(logrus.New()),
		Request:     request,
		Response:    response,
		HTTP:        &contexts.HTTPContext{},
		Integration: integrationCtx,
	})

	require.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "error", integrationCtx.State)
	assert.Empty(t, integrationCtx.Secrets[OAuthAccessTokenSecret].Value)
	assert.Empty(t, integrationCtx.Secrets[OAuthRefreshTokenSecret].Value)
	require.NotNil(t, integrationCtx.BrowserAction)

	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.Empty(t, metadata.InstallationID)
	assert.Nil(t, metadata.Organization)
	assert.Empty(t, metadata.Projects)
}

func computeWebhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
