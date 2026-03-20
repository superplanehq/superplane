package elastic

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Elastic__Sync__Validation(t *testing.T) {
	integration := &Elastic{}
	integrationCtx := &contexts.IntegrationContext{}

	t.Run("missing authType", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"url":       "https://elastic.example.com",
				"kibanaUrl": "https://kibana.example.com",
			},
			Integration: integrationCtx,
			HTTP:        &contexts.HTTPContext{},
		})
		require.ErrorContains(t, err, "authType is required")
	})

	t.Run("apiKey auth without apiKey", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"url":       "https://elastic.example.com",
				"kibanaUrl": "https://kibana.example.com",
				"authType":  "apiKey",
			},
			Integration: integrationCtx,
			HTTP:        &contexts.HTTPContext{},
		})
		require.ErrorContains(t, err, "apiKey is required when authType is apiKey")
	})

	t.Run("basic auth without username", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"url":       "https://elastic.example.com",
				"kibanaUrl": "https://kibana.example.com",
				"authType":  "basic",
				"password":  "secret",
			},
			Integration: integrationCtx,
			HTTP:        &contexts.HTTPContext{},
		})
		require.ErrorContains(t, err, "username is required when authType is basic")
	})

	t.Run("unknown authType", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"url":       "https://elastic.example.com",
				"kibanaUrl": "https://kibana.example.com",
				"authType":  "oauth",
			},
			Integration: integrationCtx,
			HTTP:        &contexts.HTTPContext{},
		})
		require.ErrorContains(t, err, `unknown authType "oauth"`)
	})
}

func Test__Elastic__Sync__Ready(t *testing.T) {
	integration := &Elastic{}

	t.Run("apiKey auth marks integration ready", func(t *testing.T) {
		config := map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"authType":  "apiKey",
			"apiKey":    "test-api-key",
		}

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{Configuration: config}

		err := integration.Sync(core.SyncContext{
			Configuration: config,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "ApiKey test-api-key", httpCtx.Requests[0].Header.Get("Authorization"))
		assert.Equal(t, "https://elastic.example.com/", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "https://kibana.example.com/api/actions/connectors", httpCtx.Requests[1].URL.String())
	})

	t.Run("basic auth marks integration ready", func(t *testing.T) {
		config := map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
			"authType":  "basic",
			"username":  "elastic",
			"password":  "secret",
		}

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{Configuration: config}

		err := integration.Sync(core.SyncContext{
			Configuration: config,
			Integration:   integrationCtx,
			HTTP:          httpCtx,
		})
		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpCtx.Requests, 2)
		user, pass, ok := httpCtx.Requests[0].BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "elastic", user)
		assert.Equal(t, "secret", pass)
	})
}
