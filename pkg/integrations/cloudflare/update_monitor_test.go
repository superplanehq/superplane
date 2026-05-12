package cloudflare

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateMonitor__Setup(t *testing.T) {
	component := &UpdateMonitor{}

	t.Run("missing monitor returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "monitor is required")
	})

	t.Run("invalid type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
				"type":    "ftp",
			},
		})

		require.ErrorContains(t, err, "type must be one of")
	})

	t.Run("invalid port returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
				"port":    99999,
			},
		})

		require.ErrorContains(t, err, "port must be between 1 and 65535")
	})

	t.Run("advanced interval below minimum returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
				"advanced": map[string]any{
					"interval": 5,
					"timeout":  3,
				},
			},
		})

		require.ErrorContains(t, err, "interval must be at least")
	})

	t.Run("advanced timeout only uses fetched monitor interval for relationship check", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"success":true,"result":{"id":"monitor123","description":"LB","interval":120,"timeout":5}}`,
					)),
				},
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
				"advanced": map[string]any{
					"timeout": 70,
				},
			},
			HTTP:     httpContext,
			Metadata: &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "token123",
					"accountId": "account123",
				},
			},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
	})

	t.Run("advanced timeout only passes without integration when relationship cannot be checked", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
				"advanced": map[string]any{
					"timeout": 70,
				},
			},
		})

		require.NoError(t, err)
	})

	t.Run("advanced timeout rejected when it is not less than fetched monitor interval", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"success":true,"result":{"id":"monitor123","description":"LB","interval":60,"timeout":5}}`,
					)),
				},
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
				"advanced": map[string]any{
					"timeout": 70,
				},
			},
			HTTP:     httpContext,
			Metadata: &contexts.MetadataContext{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "token123",
					"accountId": "account123",
				},
			},
		})

		require.ErrorContains(t, err, "timeout (70s) must be less than interval (60s)")
	})

	t.Run("validation passes without metadata resolution when integration context is absent", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
			},
		})

		require.NoError(t, err)
	})
}

func Test__UpdateMonitor__Execute(t *testing.T) {
	component := &UpdateMonitor{}

	t.Run("updates monitor by fetching current state then PUT", func(t *testing.T) {
		currentMonitor := `{
			"success": true,
			"result": {
				"id": "monitor123",
				"type": "https",
				"description": "Old name",
				"method": "GET",
				"path": "/health",
				"expected_codes": "2xx",
				"interval": 60,
				"timeout": 5,
				"retries": 0,
				"port": 443
			}
		}`
		updatedMonitor := `{
			"success": true,
			"result": {
				"id": "monitor123",
				"type": "https",
				"description": "New name",
				"method": "GET",
				"path": "/health",
				"expected_codes": "2xx",
				"interval": 60,
				"timeout": 5,
				"retries": 0,
				"port": 443
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(currentMonitor))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(updatedMonitor))},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"monitor":     "monitor123",
				"description": "New name",
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
		assert.Equal(t, UpdateMonitorPayloadType, execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/accounts/account123/load_balancers/monitors/monitor123",
			httpContext.Requests[0].URL.String(),
		)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t,
			"https://api.cloudflare.com/client/v4/accounts/account123/load_balancers/monitors/monitor123",
			httpContext.Requests[1].URL.String(),
		)
		assert.Equal(t, http.MethodPut, httpContext.Requests[1].Method)

		var body map[string]any
		require.NoError(t, json.NewDecoder(httpContext.Requests[1].Body).Decode(&body))
		assert.Equal(t, "New name", body["description"])
		assert.Equal(t, "https", body["type"])
	})

	t.Run("preserves current fields not specified in spec", func(t *testing.T) {
		currentMonitor := `{
			"success": true,
			"result": {
				"id": "monitor123",
				"type": "https",
				"description": "Old name",
				"path": "/original",
				"expected_codes": "2xx",
				"interval": 60,
				"timeout": 5
			}
		}`
		updatedMonitor := `{"success": true, "result": {"id": "monitor123", "type": "https", "description": "Old name", "path": "/new-path"}}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(currentMonitor))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(updatedMonitor))},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		newPath := "/new-path"
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"monitor": "monitor123",
				"path":    newPath,
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

		var body map[string]any
		require.NoError(t, json.NewDecoder(httpContext.Requests[1].Body).Decode(&body))
		assert.Equal(t, "/new-path", body["path"])
		assert.Equal(t, "Old name", body["description"])
	})

	t.Run("API error on fetch returns error", func(t *testing.T) {
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
		assert.Contains(t, err.Error(), "failed to fetch current monitor")
	})
}
