package cloudflare

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteMonitor__Setup(t *testing.T) {
	component := &DeleteMonitor{}

	t.Run("missing monitor returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "monitor is required")
	})

	t.Run("validation passes without metadata resolution when integration context is absent", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
			},
		})

		require.NoError(t, err)
	})

	t.Run("resolves monitor metadata when integration is available", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"success":true,"result":{"id":"monitor123","description":"Edge health"}}`,
					)),
				},
			},
		}

		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
			},
			HTTP:     httpContext,
			Metadata: metadata,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "token123",
					"accountId": "account123",
				},
			},
		})

		require.NoError(t, err)

		var meta MonitorNodeMetadata
		require.NoError(t, mapstructure.Decode(metadata.Metadata, &meta))
		assert.Equal(t, "monitor123", meta.MonitorID)
		assert.Equal(t, "Edge health", meta.MonitorDescription)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(
			t,
			"https://api.cloudflare.com/client/v4/accounts/account123/load_balancers/monitors/monitor123",
			httpContext.Requests[0].URL.String(),
		)
	})
}

func Test__DeleteMonitor__Execute(t *testing.T) {
	component := &DeleteMonitor{}

	t.Run("refuses to delete referenced monitor without force", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": [
								{"resource_type": "pool", "resource_id": "pool123", "resource_name": "Production"}
							]
						}
					`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "token123",
					"accountId": "account123",
				},
			},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "set force to delete anyway")
		require.Len(t, httpContext.Requests, 1)
	})

	t.Run("deletes unreferenced monitor", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": []}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": {"id": "monitor123"}}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "token123",
					"accountId": "account123",
				},
			},
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.Equal(t, DeleteMonitorPayloadType, execState.Type)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/account123/load_balancers/monitors/monitor123/references", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/account123/load_balancers/monitors/monitor123", httpContext.Requests[1].URL.String())
	})
}
