package cloudflare

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateMonitor__Setup(t *testing.T) {
	component := &CreateMonitor{}

	t.Run("missing type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "type is required")
	})

	t.Run("http monitor requires path", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"type":        "https",
				"description": "API health",
				"port":        443,
			},
		})

		require.ErrorContains(t, err, "path is required")
	})

	t.Run("tcp monitor requires port", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"type":        "tcp",
				"description": "API health",
			},
		})

		require.ErrorContains(t, err, "port is required")
	})

	t.Run("stale advanced timing is ignored", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"type":        "https",
				"description": "API health",
				"path":        "/health",
				"port":        443,
				"advanced": map[string]any{
					"interval": 1,
					"timeout":  5,
					"retries":  2,
				},
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid https monitor passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"type":        "https",
				"description": "API health",
				"path":        "/health",
				"port":        443,
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateMonitor__Configuration(t *testing.T) {
	component := &CreateMonitor{}
	fields := component.Configuration()

	requireField := func(name string) configuration.Field {
		t.Helper()
		for _, field := range fields {
			if field.Name == name {
				return field
			}
		}

		t.Fatalf("field %s not found", name)
		return configuration.Field{}
	}

	path := requireField("path")
	assert.Equal(t, "/", path.Default)
	require.Len(t, path.RequiredConditions, 1)
	assert.Equal(t, "type", path.RequiredConditions[0].Field)
	assert.Equal(t, httpMonitorTypes, path.RequiredConditions[0].Values)

	port := requireField("port")
	require.Len(t, port.RequiredConditions, 1)
	assert.Equal(t, "type", port.RequiredConditions[0].Field)
	assert.Equal(t, portMonitorTypes, port.RequiredConditions[0].Values)

	advanced := requireField("advanced")
	assert.True(t, advanced.Togglable)
	require.NotNil(t, advanced.TypeOptions)
	require.NotNil(t, advanced.TypeOptions.Object)
	assert.NotEmpty(t, advanced.TypeOptions.Object.Schema)
	for _, fieldName := range []string{"interval", "timeout", "retries"} {
		for _, field := range advanced.TypeOptions.Object.Schema {
			assert.NotEqual(t, fieldName, field.Name)
		}
	}

	for _, fieldName := range []string{"interval", "timeout", "retries"} {
		for _, field := range fields {
			assert.NotEqual(t, fieldName, field.Name)
		}
	}
}

func Test__CreateMonitor__CreateMonitorRequest(t *testing.T) {
	req := createMonitorRequest(CreateMonitorSpec{
		Type:        "https",
		Description: "API health",
		Path:        "/health",
		Port:        func() *int { port := 443; return &port }(),
	})

	assert.Equal(t, "https", req.Type)
	assert.Equal(t, "API health", req.Description)
	assert.Equal(t, "/health", req.Path)
	require.NotNil(t, req.Port)
	assert.Equal(t, 443, *req.Port)
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "2xx", req.ExpectedCodes)
	require.NotNil(t, req.Interval)
	assert.Equal(t, 1, *req.Interval)
	require.NotNil(t, req.Timeout)
	assert.Equal(t, 1, *req.Timeout)
	require.NotNil(t, req.Retries)
	assert.Equal(t, 0, *req.Retries)
	require.NotNil(t, req.FollowRedirects)
	assert.True(t, *req.FollowRedirects)

	interval := 60
	timeout := 5
	retries := 2
	req = createMonitorRequest(CreateMonitorSpec{
		Type:        "https",
		Description: "API health",
		Path:        "/health",
		Port:        func() *int { port := 443; return &port }(),
		Advanced: &CreateMonitorAdvancedSpec{
			Interval: &interval,
			Timeout:  &timeout,
			Retries:  &retries,
		},
	})

	require.NotNil(t, req.Interval)
	assert.Equal(t, 1, *req.Interval)
	require.NotNil(t, req.Timeout)
	assert.Equal(t, 1, *req.Timeout)
	require.NotNil(t, req.Retries)
	assert.Equal(t, 0, *req.Retries)
}

func Test__CreateMonitor__Execute(t *testing.T) {
	component := &CreateMonitor{}

	t.Run("creates monitor and attaches it to pool", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "monitor123",
								"type": "https",
								"description": "API health",
								"path": "/health",
								"method": "GET",
								"expected_codes": "2xx"
							}
						}
					`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "pool123",
								"name": "Production",
								"monitor": "monitor123"
							}
						}
					`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"type":          "https",
				"description":   "API health",
				"method":        "GET",
				"path":          "/health",
				"port":          443,
				"expectedCodes": "2xx",
				"pool":          "pool123",
				"headers": []map[string]any{
					{"name": "Host", "value": "api.example.com"},
				},
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
		assert.Equal(t, MonitorPayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/account123/load_balancers/monitors", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/account123/load_balancers/pools/pool123", httpContext.Requests[1].URL.String())

		var body map[string]any
		require.NoError(t, json.NewDecoder(httpContext.Requests[0].Body).Decode(&body))
		assert.Equal(t, "https", body["type"])
		assert.Equal(t, "/health", body["path"])
		assert.Equal(t, "2xx", body["expected_codes"])
		assert.Equal(t, map[string]any{"Host": []any{"api.example.com"}}, body["header"])
	})
}
