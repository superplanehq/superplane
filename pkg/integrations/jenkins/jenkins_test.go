package jenkins

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func validConfig() map[string]any {
	return map[string]any{
		"baseUrl":  "https://jenkins.example.com",
		"username": "admin",
		"apiToken": "token-123",
	}
}

func Test__Jenkins__Sync(t *testing.T) {
	j := &Jenkins{}

	t.Run("success verifying connection -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"mode":"NORMAL"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: validConfig()}

		err := j.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://jenkins.example.com/api/json", httpContext.Requests[0].URL.String())

		username, password, ok := httpContext.Requests[0].BasicAuth()
		require.True(t, ok, "expected basic auth header to be set")
		assert.Equal(t, "admin", username)
		assert.Equal(t, "token-123", password)
	})

	t.Run("basic auth header is base64-encoded username:apiToken", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: validConfig()}

		err := j.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		expected := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:token-123"))
		assert.Equal(t, expected, httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("401 response -> clean auth error, not raw body", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body: io.NopCloser(strings.NewReader(
						`<html><body><h2>HTTP ERROR 401 Unauthorized</h2></body></html>`,
					)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: validConfig()}

		err := j.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed (401)")
		assert.NotContains(t, err.Error(), "<html>")
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("non-200, non-401 response -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`internal error`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{Configuration: validConfig()}

		err := j.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
	})

	t.Run("missing baseUrl -> config error, no request made", func(t *testing.T) {
		config := validConfig()
		delete(config, "baseUrl")

		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{Configuration: config}

		err := j.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("missing username -> config error, no request made", func(t *testing.T) {
		config := validConfig()
		delete(config, "username")

		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{Configuration: config}

		err := j.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("missing apiToken -> config error, no request made", func(t *testing.T) {
		config := validConfig()
		delete(config, "apiToken")

		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{Configuration: config}

		err := j.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", integrationCtx.State)
		assert.Empty(t, httpContext.Requests)
	})
}
