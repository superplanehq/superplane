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

	t.Run("advanced interval below Cloudflare minimum fails", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"type":        "https",
				"description": "API health",
				"path":        "/health",
				"port":        443,
				"advanced": map[string]any{
					"interval": 9,
					"timeout":  5,
				},
			},
		})

		require.ErrorContains(t, err, "interval must be at least")
	})

	t.Run("advanced timeout must be less than interval", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"type":        "https",
				"description": "API health",
				"path":        "/health",
				"port":        443,
				"advanced": map[string]any{
					"interval": 60,
					"timeout":  60,
				},
			},
		})

		require.ErrorContains(t, err, "timeout")
		require.ErrorContains(t, err, "must be less than interval")
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

	t.Run("optional pool resolves pool name metadata when accountId is configured", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"type":        "https",
				"description": "API health",
				"path":        "/health",
				"port":        443,
				"pool":        "pool123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"success": true,
							"result": {"id": "pool123", "name": "my-pool"}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken":  "token123",
					"accountId": "account123",
				},
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)

		meta, ok := ctx.Metadata.(*contexts.MetadataContext)
		require.True(t, ok)
		poolMeta, ok := meta.Metadata.(PoolNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "my-pool", poolMeta.PoolName)
	})

	t.Run("pool without accountId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"type":        "https",
				"description": "API health",
				"path":        "/health",
				"port":        443,
				"pool":        "pool123",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "token123"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "accountId is required")
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

	names := map[string]bool{}
	for _, field := range advanced.TypeOptions.Object.Schema {
		names[field.Name] = true
	}
	for _, fieldName := range []string{"interval", "timeout", "retries"} {
		assert.True(t, names[fieldName], "advanced schema should include %s", fieldName)
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
	assert.Nil(t, req.Interval)
	assert.Nil(t, req.Timeout)
	assert.Nil(t, req.Retries)
	require.NotNil(t, req.FollowRedirects)
	assert.True(t, *req.FollowRedirects)

	interval := 120
	timeout := 8
	retries := 3
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
	assert.Equal(t, interval, *req.Interval)
	require.NotNil(t, req.Timeout)
	assert.Equal(t, timeout, *req.Timeout)
	require.NotNil(t, req.Retries)
	assert.Equal(t, retries, *req.Retries)
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
				"type":        "https",
				"description": "API health",
				"path":        "/health",
				"port":        443,
				"pool":        "pool123",
				"advanced": map[string]any{
					"method":        "GET",
					"expectedCodes": "2xx",
					"headers": []map[string]any{
						{"name": "Host", "value": "api.example.com"},
					},
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
		assert.Equal(t, CreateMonitorPayloadType, execState.Type)
		require.Len(t, execState.Payloads, 1)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/account123/load_balancers/monitors", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/account123/load_balancers/pools/pool123", httpContext.Requests[1].URL.String())

		var body map[string]any
		require.NoError(t, json.NewDecoder(httpContext.Requests[0].Body).Decode(&body))
		assert.Equal(t, "https", body["type"])
		assert.Equal(t, "/health", body["path"])
		assert.Equal(t, "2xx", body["expected_codes"])
		assert.NotContains(t, body, "interval")
		assert.NotContains(t, body, "timeout")
		assert.NotContains(t, body, "retries")
		assert.Equal(t, map[string]any{"Host": []any{"api.example.com"}}, body["header"])
	})
}

func Test__effectiveMonitorAdvanced__mergesFlatLegacyFields(t *testing.T) {
	t.Run("partial advanced inherits flat consecutive thresholds", func(t *testing.T) {
		cu := 2
		cd := 4
		adv := effectiveMonitorAdvanced(CreateMonitorSpec{
			Advanced: &CreateMonitorAdvancedSpec{
				Interval: func() *int { i := 60; return &i }(),
				Timeout:  func() *int { t := 5; return &t }(),
			},
			ConsecutiveUp:   &cu,
			ConsecutiveDown: &cd,
		})
		require.NotNil(t, adv.Interval)
		require.NotNil(t, adv.ConsecutiveUp)
		require.NotNil(t, adv.ConsecutiveDown)
		assert.Equal(t, 60, *adv.Interval)
		assert.Equal(t, 2, *adv.ConsecutiveUp)
		assert.Equal(t, 4, *adv.ConsecutiveDown)
	})

	t.Run("advanced timing overrides flat timing when both set", func(t *testing.T) {
		adv := effectiveMonitorAdvanced(CreateMonitorSpec{
			Interval: func() *int { i := 999; return &i }(),
			Advanced: &CreateMonitorAdvancedSpec{
				Interval: func() *int { i := 120; return &i }(),
			},
		})
		require.NotNil(t, adv.Interval)
		assert.Equal(t, 120, *adv.Interval)
	})

	t.Run("flat fields used when advanced is nil", func(t *testing.T) {
		adv := effectiveMonitorAdvanced(CreateMonitorSpec{
			Interval: func() *int { i := 30; return &i }(),
			Retries:  func() *int { i := 1; return &i }(),
		})
		require.NotNil(t, adv.Interval)
		require.NotNil(t, adv.Retries)
		assert.Equal(t, 30, *adv.Interval)
		assert.Equal(t, 1, *adv.Retries)
	})
}

func Test__decodeCreateMonitorSpec__flatConsecutiveWithPartialAdvanced(t *testing.T) {
	spec, err := decodeCreateMonitorSpec(map[string]any{
		"type":          "https",
		"description":   "API health",
		"path":          "/health",
		"port":          float64(443),
		"consecutiveUp": float64(2),
		"advanced": map[string]any{
			"interval": float64(60),
			"timeout":  float64(5),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, spec.ConsecutiveUp)
	assert.Equal(t, 2, *spec.ConsecutiveUp)

	req := createMonitorRequest(spec)
	require.NotNil(t, req.ConsecutiveUp)
	assert.Equal(t, 2, *req.ConsecutiveUp)
	require.NotNil(t, req.Interval)
	assert.Equal(t, 60, *req.Interval)
}

func Test__decodeCreateMonitorSpec__weakNumericTypes(t *testing.T) {
	spec, err := decodeCreateMonitorSpec(map[string]any{
		"type":        "https",
		"description": "API health",
		"path":        "/health",
		"port":        float64(443),
		"advanced": map[string]any{
			"interval": float64(16),
			"timeout":  float64(5),
			"retries":  float64(2),
		},
	})

	require.NoError(t, err)
	require.NotNil(t, spec.Advanced)
	require.NotNil(t, spec.Advanced.Interval)
	require.NotNil(t, spec.Advanced.Timeout)
	require.NotNil(t, spec.Advanced.Retries)
	assert.Equal(t, 16, *spec.Advanced.Interval)
	assert.Equal(t, 5, *spec.Advanced.Timeout)
	assert.Equal(t, 2, *spec.Advanced.Retries)
}

func Test__augmentLoadBalancerMonitorCreateError(t *testing.T) {
	apiErr := newCloudflareAPIError(http.StatusBadRequest, []byte(
		`{"success":false,"errors":[{"code":1002,"message":"interval is not in range [1, 1]: validation failed"}]}`,
	))

	out := augmentLoadBalancerMonitorCreateError(apiErr)
	require.ErrorIs(t, out, apiErr)
	require.Contains(t, out.Error(), "plan-specific")
}
