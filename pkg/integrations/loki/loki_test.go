package loki

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

func Test__Loki__Sync(t *testing.T) {
	l := &Loki{}

	t.Run("no baseURL -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "",
				"authType": AuthTypeNone,
			},
		}

		err := l.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "baseURL is required")
	})

	t.Run("no authType -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": "",
			},
		}

		err := l.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "authType is required")
	})

	t.Run("successful connection with no auth -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("ready")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
			},
		}

		err := l.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ready")
	})

	t.Run("successful connection with basic auth -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("ready")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeBasic,
				"username": "admin",
				"password": "secret",
			},
		}

		err := l.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)

		username, password, ok := httpContext.Requests[0].BasicAuth()
		assert.True(t, ok)
		assert.Equal(t, "admin", username)
		assert.Equal(t, "secret", password)
	})

	t.Run("successful connection with bearer auth -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("ready")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":     "https://loki.example.com",
				"authType":    AuthTypeBearer,
				"bearerToken": "my-token",
			},
		}

		err := l.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "Bearer my-token", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("connection with tenant ID sends X-Scope-OrgID header", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("ready")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
				"tenantID": "my-tenant",
			},
		}

		err := l.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "my-tenant", httpContext.Requests[0].Header.Get("X-Scope-OrgID"))
	})

	t.Run("connection failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusServiceUnavailable,
					Body:       io.NopCloser(strings.NewReader("Loki is not ready")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
			},
		}

		err := l.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify connection")
		assert.NotEqual(t, "ready", appCtx.State)
	})
}

func Test__Loki__Components(t *testing.T) {
	l := &Loki{}
	components := l.Components()

	require.Len(t, components, 2)
	assert.Equal(t, "loki.pushLogs", components[0].Name())
	assert.Equal(t, "loki.queryLogs", components[1].Name())
}

func Test__Loki__Triggers(t *testing.T) {
	l := &Loki{}
	triggers := l.Triggers()

	require.Len(t, triggers, 0)
}

func Test__Loki__Configuration(t *testing.T) {
	l := &Loki{}
	config := l.Configuration()

	require.Len(t, config, 6)

	baseURLField := config[0]
	assert.Equal(t, "baseURL", baseURLField.Name)
	assert.True(t, baseURLField.Required)

	authTypeField := config[1]
	assert.Equal(t, "authType", authTypeField.Name)
	assert.True(t, authTypeField.Required)

	usernameField := config[2]
	assert.Equal(t, "username", usernameField.Name)
	assert.False(t, usernameField.Required)
	assert.Len(t, usernameField.VisibilityConditions, 1)

	passwordField := config[3]
	assert.Equal(t, "password", passwordField.Name)
	assert.True(t, passwordField.Sensitive)
	assert.Len(t, passwordField.VisibilityConditions, 1)

	bearerTokenField := config[4]
	assert.Equal(t, "bearerToken", bearerTokenField.Name)
	assert.True(t, bearerTokenField.Sensitive)
	assert.Len(t, bearerTokenField.VisibilityConditions, 1)

	tenantIDField := config[5]
	assert.Equal(t, "tenantID", tenantIDField.Name)
	assert.False(t, tenantIDField.Required)
}

func Test__Loki__Instructions(t *testing.T) {
	l := &Loki{}
	instructions := l.Instructions()

	assert.NotEmpty(t, instructions)
	assert.Contains(t, instructions, "Loki URL")
	assert.Contains(t, instructions, "Authentication")
}
