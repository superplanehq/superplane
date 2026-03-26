package sentry

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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
		assert.Equal(t, "error", integrationCtx.State)
		assert.Contains(t, integrationCtx.StateDescription, "missing User Token")
		assert.Contains(t, integrationCtx.StateDescription, "Client Secret")
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, "https://sentry.io/settings/", integrationCtx.BrowserAction.URL)
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Internal Integration")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Create a [personal auth token]")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Integration Name")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "SuperPlane")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Create New Integration")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "User Token")
		assert.Contains(t, integrationCtx.BrowserAction.Description, SentryPersonalTokensURL)
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Settings → Developer Settings → Custom Integrations")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Token Permissions")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Issue & Event -> Read & Write")
	})

	t.Run("missing credentials overrides previously ready state", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			State:         "ready",
			Configuration: map[string]any{
				"baseUrl":      "https://sentry.io",
				"clientSecret": "client-secret",
			},
		}

		err := impl.Sync(core.SyncContext{
			Configuration:   integrationCtx.Configuration,
			Integration:     integrationCtx,
			BaseURL:         "https://app.example.com",
			WebhooksBaseURL: "https://hooks.example.com",
		})

		require.NoError(t, err)
		assert.Equal(t, "error", integrationCtx.State)
		assert.Equal(t, "Sentry configuration is incomplete: missing User Token", integrationCtx.StateDescription)
		require.NotNil(t, integrationCtx.BrowserAction)
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
			Subscriptions: []contexts.Subscription{
				{Configuration: SubscriptionConfiguration{Resources: []string{"issue"}}},
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
			Subscriptions: []contexts.Subscription{
				{Configuration: SubscriptionConfiguration{Resources: []string{"issue"}}},
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
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Webhook URL")
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
			Subscriptions: []contexts.Subscription{
				{Configuration: SubscriptionConfiguration{Resources: []string{"issue"}}},
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
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Webhook Subscriptions")
	})

	t.Run("integration name match ignores case", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{
				"baseUrl":         "https://sentry.io",
				"integrationName": "SuperPlane",
				"userToken":       "auth-token",
				"clientSecret":    "client-secret",
			},
			Subscriptions: []contexts.Subscription{
				{Configuration: SubscriptionConfiguration{Resources: []string{"issue"}}},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[{"id":"1","slug":"example","name":"Example Org"}]`),
				sentryMockResponse(http.StatusOK, `{"id":"1","slug":"example","name":"Example Org"}`),
				sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"backend","name":"Backend"}]`),
				sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
				sentryMockResponse(http.StatusOK, `[{"name":"Superplane","slug":"superplane","scopes":["org:read","org:write","project:read","team:read","event:read","event:write"],"events":[],"webhookUrl":"","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[]}]`),
				sentryMockResponse(http.StatusOK, `{"name":"Superplane","slug":"superplane","scopes":["org:read","org:write","project:read","team:read","event:read","event:write"],"events":["issue"],"webhookUrl":"https://hooks.example.com/api/v1/integrations/8f5fbc57-2738-409a-a6f8-af65c2de733c/events","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[]}`),
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
	})

	t.Run("missing named integration shows explicit fix instructions", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
			Configuration: map[string]any{
				"baseUrl":         "https://sentry.io",
				"integrationName": "SuperPlane",
				"userToken":       "auth-token",
				"clientSecret":    "client-secret",
			},
			Subscriptions: []contexts.Subscription{
				{Configuration: SubscriptionConfiguration{Resources: []string{"issue"}}},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[{"id":"1","slug":"example","name":"Example Org"}]`),
				sentryMockResponse(http.StatusOK, `{"id":"1","slug":"example","name":"Example Org"}`),
				sentryMockResponse(http.StatusOK, `[{"id":"2","slug":"backend","name":"Backend"}]`),
				sentryMockResponse(http.StatusOK, `[{"id":"3","slug":"platform","name":"Platform"}]`),
				sentryMockResponse(http.StatusOK, `[{"name":"Another App","slug":"another-app","scopes":["org:read"],"events":[],"webhookUrl":"","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[]}]`),
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
		assert.Equal(t, "error", integrationCtx.State)
		assert.Contains(t, integrationCtx.StateDescription, `could not find an internal integration named "SuperPlane"`)
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Contains(t, integrationCtx.BrowserAction.Description, "could not find an internal integration named `SuperPlane`")
		assert.Contains(t, integrationCtx.BrowserAction.Description, "update **Integration Name** in SuperPlane")
		assert.NotContains(t, integrationCtx.BrowserAction.Description, "Webhook URL")
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
			Subscriptions: []contexts.Subscription{
				{Configuration: SubscriptionConfiguration{Resources: []string{"issue"}}},
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
		assert.Contains(t, integrationCtx.BrowserAction.Description, "Webhook Subscriptions")
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
			Subscriptions: []contexts.Subscription{
				{Configuration: SubscriptionConfiguration{Resources: []string{"issue"}}},
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

	t.Run("no subscriptions -> still configures webhook", func(t *testing.T) {
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
				sentryMockResponse(http.StatusOK, `[{"name":"SuperPlane","slug":"superplane","scopes":["org:read"],"events":[],"webhookUrl":"","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[]}]`),
				sentryMockResponse(http.StatusOK, `{"name":"SuperPlane","slug":"superplane","scopes":["org:read","event:read"],"events":["issue"],"webhookUrl":"https://hooks.example.com/api/v1/integrations/8f5fbc57-2738-409a-a6f8-af65c2de733c/events","isInternal":true,"isAlertable":false,"verifyInstall":false,"allowedOrigins":[]}`),
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
		require.Len(t, httpContext.Requests, 6)
		assert.Equal(t, "https://sentry.io/api/0/sentry-apps/superplane/", httpContext.Requests[5].URL.String())
	})
}

func Test__Sentry__NormalizeBaseURL(t *testing.T) {
	t.Run("trims leading and trailing whitespace before scheme normalization", func(t *testing.T) {
		assert.Equal(t, "https://sentry.io", normalizeBaseURL("  https://sentry.io/  "))
	})

	t.Run("adds https after trimming whitespace when scheme is missing", func(t *testing.T) {
		assert.Equal(t, "https://washington-x2.sentry.io", normalizeBaseURL("  washington-x2.sentry.io/  "))
	})

	t.Run("uses default base URL when value is only whitespace", func(t *testing.T) {
		assert.Equal(t, DefaultBaseURL, normalizeBaseURL("   "))
	})
}

func Test__Sentry__NewClient_TrimsBaseURLWhitespace(t *testing.T) {
	client, err := NewClient(
		&contexts.HTTPContext{},
		&contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":   "  https://washington-x2.sentry.io/  ",
				"userToken": "user-token",
			},
			Metadata: Metadata{
				Organization: &OrganizationSummary{
					Slug: "washington-x2",
				},
			},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "https://washington-x2.sentry.io", client.baseURL)
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

func Test__Sentry__ListResources(t *testing.T) {
	impl := &Sentry{}

	t.Run("lists issues from the connected organization", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":   "https://sentry.io",
				"userToken": "user-token",
			},
			Metadata: Metadata{
				Organization: &OrganizationSummary{
					Slug: "example",
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[{"id":"123","shortId":"EXAMPLE-1","title":"RuntimeError: API error"},{"id":"456","shortId":"EXAMPLE-2","title":"Worker timeout"}]`),
			},
		}

		resources, err := impl.ListResources(ResourceTypeIssue, core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, []core.IntegrationResource{
			{Type: ResourceTypeIssue, ID: "123", Name: "EXAMPLE-1 · API error"},
			{Type: ResourceTypeIssue, ID: "456", Name: "EXAMPLE-2 · Worker timeout"},
		}, resources)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://sentry.io/api/0/organizations/example/issues/?query=&limit=100", httpContext.Requests[0].URL.String())
	})

	t.Run("lists assignees for the selected issue project", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":   "https://sentry.io",
				"userToken": "user-token",
			},
			Metadata: Metadata{
				Organization: &OrganizationSummary{
					Slug: "example",
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `{"id":"123","shortId":"EXAMPLE-1","title":"RuntimeError: API error","project":{"id":"10","slug":"backend","name":"Backend"}}`),
				sentryMockResponse(http.StatusOK, `[{"id":"7","name":"Alice Jones","email":"alice@example.com","user":{"id":"7","name":"Alice Jones","email":"alice@example.com","username":"alice"}}]`),
				sentryMockResponse(http.StatusOK, `[{"id":"42","slug":"platform","name":"Platform"}]`),
			},
		}

		resources, err := impl.ListResources(ResourceTypeAssignee, core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
			Parameters: map[string]string{
				"issueId": "123",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, []core.IntegrationResource{
			{Type: ResourceTypeAssignee, ID: "user:7", Name: "User · Alice Jones"},
			{Type: ResourceTypeAssignee, ID: "team:42", Name: "Team · Platform"},
		}, resources)
		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, "https://sentry.io/api/0/organizations/example/issues/123/", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://sentry.io/api/0/projects/example/backend/members/", httpContext.Requests[1].URL.String())
		assert.Equal(t, "https://sentry.io/api/0/projects/example/backend/teams/", httpContext.Requests[2].URL.String())
	})

	t.Run("lists alerts for the selected project", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":   "https://sentry.io",
				"userToken": "user-token",
			},
			Metadata: Metadata{
				Organization: &OrganizationSummary{
					Slug: "example",
				},
			},
		}

		firstPage := sentryMockResponse(
			http.StatusOK,
			`[{"id":"7","name":"High error rate","projects":["backend"]},{"id":"8","name":"Latency alert","projects":["frontend"]}]`,
		)
		firstPage.Header.Set(
			"Link",
			`<https://sentry.io/api/0/organizations/example/alert-rules/?cursor=page2>; rel="next"; results="true"; cursor="page2"`,
		)
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				firstPage,
				sentryMockResponse(http.StatusOK, `[{"id":"9","name":"Cross-project issue count","projects":["backend","frontend"]}]`),
			},
		}

		resources, err := impl.ListResources(ResourceTypeAlert, core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
			Parameters: map[string]string{
				"project": "backend",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, []core.IntegrationResource{
			{Type: ResourceTypeAlert, ID: "7", Name: "High error rate · backend"},
			{Type: ResourceTypeAlert, ID: "9", Name: "Cross-project issue count"},
		}, resources)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://sentry.io/api/0/organizations/example/alert-rules/", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://sentry.io/api/0/organizations/example/alert-rules/?cursor=page2", httpContext.Requests[1].URL.String())
	})

	t.Run("lists releases for the selected project", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":   "https://sentry.io",
				"userToken": "user-token",
			},
			Metadata: Metadata{
				Organization: &OrganizationSummary{
					Slug: "example",
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `[{"version":"2026.03.25","projects":[{"slug":"backend","name":"Backend"}]},{"version":"2026.03.24","projects":[{"slug":"frontend","name":"Frontend"}]},{"version":"2026.03.23","projects":[{"slug":"backend","name":"Backend"},{"slug":"frontend","name":"Frontend"}]}]`),
			},
		}

		resources, err := impl.ListResources(ResourceTypeRelease, core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
			Parameters: map[string]string{
				"project": "backend",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, []core.IntegrationResource{
			{Type: ResourceTypeRelease, ID: "2026.03.25", Name: "2026.03.25"},
			{Type: ResourceTypeRelease, ID: "2026.03.23", Name: "2026.03.23"},
		}, resources)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://sentry.io/api/0/organizations/example/releases/", httpContext.Requests[0].URL.String())
	})

	t.Run("surfaces release scope guidance when listing releases is forbidden", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":   "https://sentry.io",
				"userToken": "user-token",
			},
			Metadata: Metadata{
				Organization: &OrganizationSummary{
					Slug: "example",
				},
			},
		}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusForbidden, `{"detail":"You do not have permission to perform this action."}`),
			},
		}

		_, err := impl.ListResources(ResourceTypeRelease, core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), releaseScope)
	})
}

func Test__Sentry__Configuration(t *testing.T) {
	impl := &Sentry{}

	fields := impl.Configuration()

	require.Len(t, fields, 4)
	assert.Equal(t, "baseUrl", fields[0].Name)
	assert.Equal(t, "", fields[0].Default)
	assert.Equal(t, "Sentry Organization URL", fields[0].Label)
	assert.Equal(t, "userToken", fields[1].Name)
	assert.Equal(t, "User Token", fields[1].Label)
	assert.Equal(t, "integrationName", fields[2].Name)
	assert.Equal(t, "Personal auth token from Sentry. Include `project:releases` if you use release actions.", fields[1].Description)
}

func Test__Sentry__Instructions(t *testing.T) {
	impl := &Sentry{}

	instructions := impl.Instructions()

	assert.Contains(t, instructions, SentryPersonalTokensURL)
	assert.Contains(t, instructions, "User Token")
	assert.Contains(t, instructions, "Token Permissions")
	assert.Contains(t, instructions, "project:releases")
	assert.Contains(t, instructions, "Issue & Event -> `Read & Write`")
	assert.Contains(t, instructions, "Settings → Developer Settings → Custom Integrations")
}

func Test__nextCursorPath__NoTrailingQuestionWhenQueryEmpty(t *testing.T) {
	// Percent-encoded path so url.Parse sets RawPath; empty query must not yield a trailing "?".
	link := `<https://sentry.io/api/0/organizations/example/alert-rules/foo%20bar>; rel="next"; results="true"`
	parsed, err := url.Parse(strings.Trim(link, "<>"))
	require.NoError(t, err)
	if parsed.RawPath == "" {
		t.Skip("url.Parse did not populate RawPath in this environment; skipping RawPath regression check")
	}

	got := nextCursorPath(link)
	require.NotEmpty(t, got, "nextCursorPath returned empty for %q", link)
	require.False(t, strings.HasSuffix(got, "?"), "nextCursorPath must not end with '?', got %q", got)
}

func computeWebhookSignature(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
