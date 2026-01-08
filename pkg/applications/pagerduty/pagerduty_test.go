package pagerduty

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Pagerduty__Sync(t *testing.T) {
	p := &PagerDuty{}

	t.Run("no region -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region": "",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("no subdomain -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region": "us",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "subdomain is required")
	})

	t.Run("no authType -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region":    "us",
				"subdomain": "example",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "authType is required")
	})

	t.Run("invalid authType -> error", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region":    "us",
				"subdomain": "example",
				"authType":  "nope",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "authType nope is not supported")
	})

	t.Run("api token -> successful service list moves app to ready and sets metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"services": [
								{"id": "PX1234567890", "name": "Test Service"}
							]
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region":    "us",
				"subdomain": "example",
				"authType":  AuthTypeAPIToken,
				"apiToken":  "token123",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.pagerduty.com/services", httpContext.Requests[0].URL.String())

		metadata := appCtx.Metadata.(Metadata)
		assert.Len(t, metadata.Services, 1)
		assert.Equal(t, "PX1234567890", metadata.Services[0].ID)
		assert.Equal(t, "Test Service", metadata.Services[0].Name)
	})

	t.Run("api token -> failed service list returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region":    "us",
				"subdomain": "example",
				"authType":  AuthTypeAPIToken,
				"apiToken":  "token123",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.pagerduty.com/services", httpContext.Requests[0].URL.String())
		assert.Nil(t, appCtx.Metadata)
	})

	t.Run("app oauth -> clientId required", func(t *testing.T) {
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region":    "us",
				"subdomain": "example",
				"authType":  AuthTypeAppOAuth,
			},
			Secrets: map[string]core.InstallationSecret{},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "clientId is required")
	})

	t.Run("app oauth -> clientSecret required", func(t *testing.T) {
		clientID := "client-123"
		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region":    "us",
				"subdomain": "example",
				"authType":  AuthTypeAppOAuth,
				"clientId":  clientID,
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "config not found: clientSecret")
	})

	t.Run("app oauth -> token request failure returns error", func(t *testing.T) {
		clientID := "client-123"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"region":       "us",
				"subdomain":    "example",
				"authType":     AuthTypeAppOAuth,
				"clientId":     clientID,
				"clientSecret": "secret-123",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.ErrorContains(t, err, "error generating access token for app: request got 502")
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://identity.pagerduty.com/oauth/token", httpContext.Requests[0].URL.String())
	})

	t.Run("app oauth -> service list failure returns error", func(t *testing.T) {
		clientID := "client-123"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"access_token": "access-123",
							"token_type": "bearer",
							"expires_in": 3600,
							"scope": "services.read"
						}
					`)),
				},
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Secrets: map[string]core.InstallationSecret{},
			Configuration: map[string]any{
				"region":       "us",
				"subdomain":    "example",
				"authType":     AuthTypeAppOAuth,
				"clientId":     clientID,
				"clientSecret": "secret-123",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", appCtx.State)
		assert.Nil(t, appCtx.Metadata)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://identity.pagerduty.com/oauth/token", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://api.pagerduty.com/services", httpContext.Requests[1].URL.String())
	})

	t.Run("app oauth -> success creates access token, moves to ready, and sets metadata", func(t *testing.T) {
		clientID := "client-123"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"access_token": "access-123",
							"token_type": "bearer",
							"expires_in": 3600,
							"scope": "services.read"
						}
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"services": [
								{"id": "PX1234567890", "name": "Test Service"}
							]
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Secrets: map[string]core.InstallationSecret{},
			Configuration: map[string]any{
				"region":       "us",
				"subdomain":    "example",
				"authType":     AuthTypeAppOAuth,
				"clientId":     clientID,
				"clientSecret": "secret-123",
			},
		}

		err := p.Sync(core.SyncContext{
			Configuration:   appCtx.Configuration,
			HTTP:            httpContext,
			AppInstallation: appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://identity.pagerduty.com/oauth/token", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://api.pagerduty.com/services", httpContext.Requests[1].URL.String())
		secret, ok := appCtx.Secrets[AppAccessToken]
		require.True(t, ok)
		assert.Equal(t, []byte("access-123"), secret.Value)

		metadata := appCtx.Metadata.(Metadata)
		assert.Len(t, metadata.Services, 1)
		assert.Equal(t, "PX1234567890", metadata.Services[0].ID)
		assert.Equal(t, "Test Service", metadata.Services[0].Name)
	})
}

func Test__PagerDuty__CompareWebhookConfig(t *testing.T) {
	p := &PagerDuty{}

	testCases := []struct {
		name        string
		configA     any
		configB     any
		expectEqual bool
		expectError bool
	}{
		{
			name: "identical events",
			configA: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different service",
			configA: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service2",
				},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "different event",
			configA: WebhookConfiguration{
				Events: []string{"incident.resolved"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "subset of events",
			configA: WebhookConfiguration{
				Events: []string{"incident.triggered", "incident.resolved"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			configB: WebhookConfiguration{
				Events: []string{"incident.triggered"},
				Filter: WebhookFilter{
					Type: "service_reference",
					ID:   "service1",
				},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"events": []string{"incident.triggered"},
				"filter": map[string]string{
					"type": "service_reference",
					"id":   "service1",
				},
			},
			configB: map[string]any{
				"events": []string{"incident.triggered"},
				"filter": map[string]string{
					"type": "service_reference",
					"id":   "service1",
				},
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: WebhookConfiguration{
				Events: []string{"incident.triggered"},
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				Events: []string{"incident.triggered"},
			},
			configB:     "invalid",
			expectEqual: false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			equal, err := p.CompareWebhookConfig(tc.configA, tc.configB)

			if tc.expectError {
				assert.Error(t, err, "expected error, but got none")
			} else {
				require.NoError(t, err, "did not expect, but got an error")
			}

			assert.Equal(t, tc.expectEqual, equal, "expected config to be equal, but they are different")
		})
	}
}
