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

	t.Run("missing clientId -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
			},
			Secrets: map[string]core.IntegrationSecret{},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "clientId is required")
	})

	t.Run("missing clientSecret -> error", func(t *testing.T) {
		clientID := "client-123"
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"instanceUrl": "https://dev12345.service-now.com",
				"clientId":    clientID,
			},
		}

		err := s.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "config not found: clientSecret")
	})

	t.Run("token exchange failure -> error", func(t *testing.T) {
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

	t.Run("validation failure after token -> error", func(t *testing.T) {
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

	t.Run("success -> stores token, ready, schedules resync", func(t *testing.T) {
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
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": [{"label": "Software", "value": "software"}]}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": [{"sys_id": "grp1", "name": "Network"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{},
			Configuration: map[string]any{
				"instanceUrl":  "https://dev12345.service-now.com",
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
		require.Len(t, httpContext.Requests, 4)
		assert.Equal(t, "https://dev12345.service-now.com/oauth_token.do", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://dev12345.service-now.com/api/now/table/incident?sysparm_limit=1", httpContext.Requests[1].URL.String())

		secret, ok := integrationCtx.Secrets[OAuthAccessToken]
		require.True(t, ok)
		assert.Equal(t, []byte("access-123"), secret.Value)

		require.Len(t, integrationCtx.ResyncRequests, 1)
		assert.Equal(t, 900*time.Second, integrationCtx.ResyncRequests[0])

		metadata, ok := integrationCtx.Metadata.(Metadata)
		require.True(t, ok)
		assert.Len(t, metadata.Categories, 1)
		assert.Len(t, metadata.AssignmentGroups, 1)
	})
}
