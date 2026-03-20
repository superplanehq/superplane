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

	t.Run("missing url", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"kibanaUrl": "https://kibana.example.com",
				"apiKey":    "test-api-key",
			},
			Integration: integrationCtx,
			HTTP:        &contexts.HTTPContext{},
		})
		require.ErrorContains(t, err, "url is required")
	})

	t.Run("missing kibanaUrl", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"url":    "https://elastic.example.com",
				"apiKey": "test-api-key",
			},
			Integration: integrationCtx,
			HTTP:        &contexts.HTTPContext{},
		})
		require.ErrorContains(t, err, "kibanaUrl is required")
	})

	t.Run("missing apiKey", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"url":       "https://elastic.example.com",
				"kibanaUrl": "https://kibana.example.com",
			},
			Integration: integrationCtx,
			HTTP:        &contexts.HTTPContext{},
		})
		require.ErrorContains(t, err, "apiKey is required")
	})
}

func Test__Elastic__Sync__Ready(t *testing.T) {
	integration := &Elastic{}

	t.Run("apiKey auth marks integration ready", func(t *testing.T) {
		config := map[string]any{
			"url":       "https://elastic.example.com",
			"kibanaUrl": "https://kibana.example.com",
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
}
