package opencost

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

func Test__GetCostAllocation__Setup(t *testing.T) {
	component := &GetCostAllocation{}

	t.Run("window is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"window": "", "aggregate": "namespace"}})
		require.ErrorContains(t, err, "window is required")
	})

	t.Run("aggregate is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"window": "1h", "aggregate": ""}})
		require.ErrorContains(t, err, "aggregate is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"window": "1h", "aggregate": "namespace"}})
		require.NoError(t, err)
	})
}

func Test__GetCostAllocation__Execute(t *testing.T) {
	component := &GetCostAllocation{}

	t.Run("allocation data is emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"production": {
								"name": "production",
								"properties": {"namespace": "production"},
								"window": {"start": "2026-01-19T00:00:00Z", "end": "2026-01-20T00:00:00Z"},
								"start": "2026-01-19T00:00:00Z",
								"end": "2026-01-20T00:00:00Z",
								"minutes": 1440,
								"cpuCost": 45.12,
								"gpuCost": 0,
								"ramCost": 28.50,
								"pvCost": 8.20,
								"networkCost": 3.50,
								"totalCost": 85.32
							}
						}]
					}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"window": "24h", "aggregate": "namespace"},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Finished)
		assert.True(t, executionCtx.Passed)
		assert.Equal(t, CostAllocationPayloadType, executionCtx.Type)
		require.Len(t, executionCtx.Payloads, 1)
		payload := executionCtx.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "24h", payload["window"])
		assert.Equal(t, "namespace", payload["aggregate"])
		assert.Equal(t, 85.32, payload["totalCost"])
		allocations := payload["allocations"].([]map[string]any)
		require.Len(t, allocations, 1)
		assert.Equal(t, "production", allocations[0]["name"])
		assert.Equal(t, 85.32, allocations[0]["totalCost"])
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error": "internal server error"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"window": "1h", "aggregate": "namespace"},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to fetch cost allocation")
	})

	t.Run("sanitizes configuration", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"staging": {
								"name": "staging",
								"properties": {"namespace": "staging"},
								"window": {"start": "2026-01-19T00:00:00Z", "end": "2026-01-20T00:00:00Z"},
								"start": "2026-01-19T00:00:00Z",
								"end": "2026-01-20T00:00:00Z",
								"minutes": 1440,
								"cpuCost": 10.0,
								"gpuCost": 0,
								"ramCost": 5.0,
								"pvCost": 2.0,
								"networkCost": 1.0,
								"totalCost": 18.0
							}
						}]
					}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"window": "  1h  ", "aggregate": "  NAMESPACE  "},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Passed)
		require.Len(t, executionCtx.Payloads, 1)
		payload := executionCtx.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "1h", payload["window"])
		assert.Equal(t, "namespace", payload["aggregate"])
	})
}
