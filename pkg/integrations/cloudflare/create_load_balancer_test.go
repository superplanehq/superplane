package cloudflare

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

func Test__CreateLoadBalancer__Setup(t *testing.T) {
	component := &CreateLoadBalancer{}

	t.Run("missing zoneId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":         "",
				"name":         "lb.example.com",
				"fallbackPool": "pool123",
				"defaultPools": []any{"pool123"},
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("missing name returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":         "zone123",
				"name":         "",
				"fallbackPool": "pool123",
				"defaultPools": []any{"pool123"},
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing fallbackPool returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":         "zone123",
				"name":         "lb.example.com",
				"fallbackPool": "",
				"defaultPools": []any{"pool123"},
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "fallbackPool is required")
	})

	t.Run("missing defaultPools returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":         "zone123",
				"name":         "lb.example.com",
				"fallbackPool": "pool123",
				"defaultPools": []any{},
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "at least one defaultPool is required")
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":         "zone123",
				"name":         "lb.example.com",
				"fallbackPool": "pool123",
				"defaultPools": []any{"pool123"},
			},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__CreateLoadBalancer__Execute(t *testing.T) {
	component := &CreateLoadBalancer{}

	t.Run("successful create emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": {
							"id": "lb123",
							"name": "lb.example.com",
							"description": "Test LB",
							"enabled": true,
							"proxied": true,
							"fallback_pool": "pool123",
							"default_pools": ["pool123"],
							"steering_policy": "random",
							"session_affinity": "none"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"zone":           "zone123",
				"name":           "lb.example.com",
				"description":    "Test LB",
				"fallbackPool":   "pool123",
				"defaultPools":   []any{"pool123"},
				"steeringPolicy": "random",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)
		require.NoError(t, err)

		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, "cloudflare.loadBalancer.created", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/load_balancers", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message": "Name already exists"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"zone":         "zone123",
				"name":         "lb.example.com",
				"fallbackPool": "pool123",
				"defaultPools": []any{"pool123"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create load balancer")
	})
}
