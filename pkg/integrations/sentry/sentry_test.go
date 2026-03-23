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

	t.Run("missing credentials -> setup prompt", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{},
		}

		err := impl.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			Integration:     integrationCtx,
			BaseURL:         "https://app.example.com",
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, "https://sentry.io/settings/", integrationCtx.BrowserAction.URL)
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Internal Integration")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "already created a Sentry internal integration")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Integration Name")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "SuperPlane")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "New Internal Integration")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "User Token")
	})

	t.Run("valid token and integration name -> ready and webhook configured", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{
				"baseUrl":         "https://sentry.io",
				"integrationName": "SuperPlane",
				"userToken":       "auth-token",
				"clientSecret":    "client-secret",
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[{"id":"1","slug":"example","name":"Example Org"}]`),
				sentryMockResponse(http.StatusOK, `{"id":"1","slug":"example","name":"Example Org"}`),
				sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"backend","name":"Backend"}]`),
				sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
				sentryMockResponse(http.StatusOK, `[{"name":"SuperPlane","slug":"superplane","scopes":["org:read","org:write","project:read","team:read","event:read","event:write"],"events":[],"webhookUrl":"","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[]}]`),
				sentryMockResponse(http.StatusOK, `{"name":"SuperPlane","slug":"superplane","scopes":["org:read","org:write","project:read","team:read","event:read","event:write"],"events":["issue"],"webhookUrl":"https://hooks.example.com/api/v1/integrations/8f5fbc57-2738-409a-a6f8-af65c2de733c/events","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[],"clientSecret":"new-rotated-secret"}`),
			},
		}

		err := impl.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			Integration:     integrationCtx,
			HTTP:            httpContext,
			Logger:          logrus.NewEntry(logrus.New()),
			BaseURL:         "https://app.example.com",
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		assert.Nil(t, integrationCtx.BrowserAction)

		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Organization)
		assert.Equal(t, "example", metadata.Organization.Slug)
		assert.Equal(t, "superplane", metadata.AppSlug)
		require.Len(t, metadata.Projects, 1)
		require.Len(t, metadata.Teams, 1)

		require.Len(t, httpContext.Requests, 6)
		assert.Equal(t, "https://sentry.io/api/0/sentry-apps/superplane/", httpContext.Requests[5].URL.String())

		// Sentry rotates the client secret on PUT — verify the new secret was stored automatically.
		secret, ok := integrationCtx.Secrets["clientSecret"]
		require.True(t, ok)
		assert.Equal(t, "new-rotated-secret", string(secret.Value))
	})

	t.Run("multiple apps without integration name -> ready with manual webhook prompt", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{
				"baseUrl":      "https://sentry.io",
				"userToken":    "auth-token",
				"clientSecret": "client-secret",
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[{"id":"1","slug":"example","name":"Example Org"}]`),
				sentryMockResponse(http.StatusOK, `{"id":"1","slug":"example","name":"Example Org"}`),
				sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"backend","name":"Backend"}]`),
				sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
				sentryMockResponse(http.StatusOK, `[{"name":"A","slug":"a","scopes":["org:read"],"events":[],"webhookUrl":"","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[]},{"name":"B","slug":"b","scopes":["org:read"],"events":[],"webhookUrl":"","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[]}]`),
			},
		}

		err := impl.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			Integration:     integrationCtx,
			HTTP:            httpContext,
			Logger:          logrus.NewEntry(logrus.New()),
			BaseURL:         "https://app.example.com",
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Contains(t, integrationCtx.BrowserAction.Description, "multiple custom integrations exist")
		assert.Equal(t, "https://sentry.io/settings/example/developer-settings/", integrationCtx.BrowserAction.URL)
	})

	t.Run("valid token and client secret without app visibility -> ready with manual webhook prompt", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{
				"baseUrl":         "https://sentry.io",
				"integrationName": "SuperPlane",
				"userToken":       "auth-token",
				"clientSecret":    "client-secret",
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[{"id":"1","slug":"example","name":"Example Org"}]`),
				sentryMockResponse(http.StatusOK, `{"id":"1","slug":"example","name":"Example Org"}`),
				sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"backend","name":"Backend"}]`),
				sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
				sentryMockResponse(http.StatusForbidden, `{"detail":"forbidden"}`),
			},
		}

		err := impl.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			Integration:     integrationCtx,
			HTTP:            httpContext,
			Logger:          logrus.NewEntry(logrus.New()),
			BaseURL:         "https://app.example.com",
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Contains(t, integrationCtx.BrowserAction.Description, "could not automatically configure the internal integration webhook")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "failed to list internal integrations for automatic webhook configuration")
	})

	t.Run("token accepted by auth but rejected by organization listing on sentry.io -> clear error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{
				"baseUrl":      "https://sentry.io",
				"userToken":    "proxy-auth-token",
				"clientSecret": "client-secret",
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusUnauthorized, `{"detail":"Invalid token"}`),
				sentryMockResponse(http.StatusOK, `{"id":"4358126","username":"spp-example","email":"spp@example.com"}`),
			},
		}

		err := impl.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
			HTTP:          httpContext,
			Logger:        logrus.NewEntry(logrus.New()),
		})

		require.NoError(t, err)
		assert.Equal(t, "error", integrationCtx.State)
		assert.Contains(t, integrationCtx.StateDescription, "token is accepted by /auth but is not authorized for organization listing")
		assert.Contains(t, integrationCtx.StateDescription, "https://your-org.sentry.io")
	})

	t.Run("org URL provided -> syncs metadata without organization listing", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{
				"baseUrl":         "https://washington-x2.sentry.io",
				"integrationName": "SuperPlane",
				"userToken":       "auth-token",
				"clientSecret":    "client-secret",
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `{"id":"1","slug":"washington-x2","name":"Washington"}`),
				sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"go-gin","name":"go-gin"}]`),
				sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"washington","name":"Washington"}]`),
				sentryMockResponse(http.StatusForbidden, `{"detail":"forbidden"}`),
			},
		}

		err := impl.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			Integration:     integrationCtx,
			HTTP:            httpContext,
			Logger:          logrus.NewEntry(logrus.New()),
			BaseURL:         "https://app.example.com",
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Organization)
		assert.Equal(t, "washington-x2", metadata.Organization.Slug)
		require.Len(t, metadata.Projects, 1)
		require.Len(t, metadata.Teams, 1)
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Contains(t, integrationCtx.BrowserAction.Description, "failed to list internal integrations for automatic webhook configuration")
		require.Len(t, httpContext.Requests, 4)
		assert.Equal(t, "https://washington-x2.sentry.io/api/0/organizations/washington-x2/", httpContext.Requests[0].URL.String())
	})

	t.Run("matching webhook and events but missing event read scope -> updates integration", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{
				"baseUrl":         "https://sentry.io",
				"integrationName": "SuperPlane",
				"userToken":       "auth-token",
				"clientSecret":    "client-secret",
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[{"id":"1","slug":"example","name":"Example Org"}]`),
				sentryMockResponse(http.StatusOK, `{"id":"1","slug":"example","name":"Example Org"}`),
				sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"backend","name":"Backend"}]`),
				sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
				sentryMockResponse(http.StatusOK, `[{"name":"SuperPlane","slug":"superplane","scopes":["org:read","org:write","project:read","team:read","event:write"],"events":["issue"],"webhookUrl":"https://hooks.example.com/api/v1/integrations/8f5fbc57-2738-409a-a6f8-af65c2de733c/events","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[]}]`),
				sentryMockResponse(http.StatusOK, `{"name":"SuperPlane","slug":"superplane","scopes":["org:read","org:write","project:read","team:read","event:read","event:write"],"events":["issue"],"webhookUrl":"https://hooks.example.com/api/v1/integrations/8f5fbc57-2738-409a-a6f8-af65c2de733c/events","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[]}`),
			},
		}

		err := impl.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			Integration:     integrationCtx,
			HTTP:            httpContext,
			Logger:          logrus.NewEntry(logrus.New()),
			BaseURL:         "https://app.example.com",
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 6)
		assert.Equal(t, "https://sentry.io/api/0/sentry-apps/superplane/", httpContext.Requests[5].URL.String())
	})
}

func Test__Sentry__HandleWebhook(t *testing.T) {
	impl := &Sentry{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl":      "https://sentry.io",
			"userToken":    "auth-token",
			"clientSecret": "client-secret",
		},
		Subscriptions: []contexts.Subscription{
			{Configuration: SubscriptionConfiguration{Resources: []string{"issue"}}},
		},
	}

	body := []byte(`{"action":"created","installation":{"uuid":"install-123"},"data":{"issue":{"id":"123"}}}`)
	signature := computeWebhookSignature("client-secret", body)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/test/events", bytes.NewReader(body))
	request.Header.Set("Sentry-Hook-Resource", "issue")
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
}

func Test__Sentry__HandleWebhook__MissingClientSecret(t *testing.T) {
	impl := &Sentry{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl":   "https://sentry.io",
			"userToken": "auth-token",
		},
	}

	body := []byte(`{"action":"created","installation":{"uuid":"install-123"},"data":{"issue":{"id":"123"}}}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/test/events", bytes.NewReader(body))
	request.Header.Set("Sentry-Hook-Resource", "issue")
	request.Header.Set("Sentry-Hook-Signature", computeWebhookSignature("", body))
	response := httptest.NewRecorder()

	impl.HandleRequest(core.HTTPRequestContext{
		Logger:      logrus.NewEntry(logrus.New()),
		Request:     request,
		Response:    response,
		HTTP:        &contexts.HTTPContext{},
		Integration: integrationCtx,
	})

	require.Equal(t, http.StatusForbidden, response.Code)
}

func computeWebhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
