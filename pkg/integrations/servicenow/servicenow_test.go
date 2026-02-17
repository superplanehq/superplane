package servicenow

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ServiceNow__Sync(t *testing.T) {
	s := &ServiceNow{}

	t.Run("no instanceUrl -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "instanceUrl is required")
	})

	t.Run("no authType -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "authType is required")
	})

	t.Run("invalid authType -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    "nope",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "authType nope is not supported")
	})

	t.Run("basic auth -> missing username -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    AuthTypeBasicAuth,
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "username is required")
	})

	t.Run("basic auth -> missing password -> error", func(t *testing.T) {
		username := "admin"
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    AuthTypeBasicAuth,
				"username":    username,
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "password is required")
	})

	t.Run("basic auth -> validation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error": "unauthorized"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    AuthTypeBasicAuth,
				"username":    "admin",
				"password":    "wrong",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error validating credentials")
		assert.NotEqual(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://dev12345.service-now.com/api/now/table/incident?sysparm_limit=1", httpContext.Requests[0].URL.String())
	})

	t.Run("basic auth -> success -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": []}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    AuthTypeBasicAuth,
				"username":    "admin",
				"password":    "password123",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://dev12345.service-now.com/api/now/table/incident?sysparm_limit=1", httpContext.Requests[0].URL.String())
	})

	t.Run("client credentials -> missing clientId -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    AuthTypeOAuth,
			},
			Secrets: map[string]core.IntegrationSecret{},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "clientId is required")
	})

	t.Run("client credentials -> missing clientSecret -> error", func(t *testing.T) {
		clientID := "client-123"
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"authType":    AuthTypeOAuth,
				"clientId":    clientID,
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "config not found: clientSecret")
	})

	t.Run("client credentials -> token exchange failure -> error", func(t *testing.T) {
		clientID := "client-123"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl":  "https://dev12345.service-now.com",
				"authType":     AuthTypeOAuth,
				"clientId":     clientID,
				"clientSecret": "secret-123",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "error generating access token: request got 502")
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://dev12345.service-now.com/oauth_token.do", httpContext.Requests[0].URL.String())
	})

	t.Run("client credentials -> validation failure after token -> error", func(t *testing.T) {
		clientID := "client-123"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "access-123",
						"token_type": "Bearer",
						"expires_in": 1800
					}`)),
				},
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{},
			Configuration: map[string]any{
				"instanceUrl":  "https://dev12345.service-now.com",
				"authType":     AuthTypeOAuth,
				"clientId":     clientID,
				"clientSecret": "secret-123",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://dev12345.service-now.com/oauth_token.do", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://dev12345.service-now.com/api/now/table/incident?sysparm_limit=1", httpContext.Requests[1].URL.String())
	})

	t.Run("client credentials -> success -> stores token, ready, schedules resync", func(t *testing.T) {
		clientID := "client-123"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"access_token": "access-123",
						"token_type": "Bearer",
						"expires_in": 1800
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": []}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{},
			Configuration: map[string]any{
				"instanceUrl":  "https://dev12345.service-now.com",
				"authType":     AuthTypeOAuth,
				"clientId":     clientID,
				"clientSecret": "secret-123",
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://dev12345.service-now.com/oauth_token.do", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://dev12345.service-now.com/api/now/table/incident?sysparm_limit=1", httpContext.Requests[1].URL.String())

		secret, ok := integrationCtx.Secrets[OAuthAccessToken]
		require.True(t, ok)
		assert.Equal(t, []byte("access-123"), secret.Value)

		require.Len(t, integrationCtx.ResyncRequests, 1)
		assert.Equal(t, 900*time.Second, integrationCtx.ResyncRequests[0])
	})
}
