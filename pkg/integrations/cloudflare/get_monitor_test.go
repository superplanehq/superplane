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

func Test__GetMonitor__Setup(t *testing.T) {
	component := &GetMonitor{}

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
	})
}

func Test__GetMonitor__Execute(t *testing.T) {
	component := &GetMonitor{}

	t.Run("successful get emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": {
							"id": "monitor123",
							"type": "https",
							"description": "Login page monitor",
							"method": "GET",
							"path": "/health",
							"expected_codes": "2xx",
							"interval": 60,
							"timeout": 5,
							"retries": 0
						}
					}`)),
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
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, GetMonitorPayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/accounts/account123/load_balancers/monitors/monitor123",
			httpContext.Requests[0].URL.String(),
		)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"success":false,"errors":[{"message":"Monitor not found"}]}`)),
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

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get monitor")
	})
}
