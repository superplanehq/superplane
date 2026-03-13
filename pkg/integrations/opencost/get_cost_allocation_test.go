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
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"window": "1d", "aggregate": ""}})
		require.ErrorContains(t, err, "aggregate is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"window":    "1d",
			"aggregate": "namespace",
		}})
		require.NoError(t, err)
	})

	t.Run("valid setup with step", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"window":    "7d",
			"aggregate": "cluster",
			"step":      "1d",
		}})
		require.NoError(t, err)
	})
}

func Test__GetCostAllocation__Execute(t *testing.T) {
	component := &GetCostAllocation{}

	t.Run("successful execution emits cost data", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [
							{
								"production": {
									"name": "production",
									"properties": {"namespace": "production"},
									"window": {"start": "2026-02-21T00:00:00Z", "end": "2026-02-22T00:00:00Z"},
									"start": "2026-02-21T00:00:00Z",
									"end": "2026-02-22T00:00:00Z",
									"cpuCost": 45.23,
									"gpuCost": 0,
									"ramCost": 38.92,
									"pvCost": 8.75,
									"networkCost": 12.5,
									"totalCost": 105.4,
									"cpuEfficiency": 0.42,
									"ramEfficiency": 0.61,
									"totalEfficiency": 0.51
								},
								"kube-system": {
									"name": "kube-system",
									"properties": {"namespace": "kube-system"},
									"window": {"start": "2026-02-21T00:00:00Z", "end": "2026-02-22T00:00:00Z"},
									"start": "2026-02-21T00:00:00Z",
									"end": "2026-02-22T00:00:00Z",
									"cpuCost": 12.1,
									"gpuCost": 0,
									"ramCost": 8.45,
									"pvCost": 0,
									"networkCost": 3.2,
									"totalCost": 23.75,
									"cpuEfficiency": 0.35,
									"ramEfficiency": 0.28,
									"totalEfficiency": 0.31
								}
							}
						]
					}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"window": "1d", "aggregate": "namespace"},
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
		assert.Equal(t, "1d", payload["window"])
		assert.Equal(t, "namespace", payload["aggregate"])

		allocations := payload["allocations"].([]map[string]any)
		require.Len(t, allocations, 2)
	})

	t.Run("execute sanitizes configuration", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"code": 200, "data": [{"default": {"name": "default", "totalCost": 10}}]}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"window": "  1d  ", "aggregate": "  NAMESPACE  "},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Passed)

		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "window=1d")
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "aggregate=namespace")
	})

	t.Run("API error fails execution", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`error`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"window": "1d", "aggregate": "namespace"},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to fetch cost allocation")
	})

	t.Run("step parameter is included when set", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"code": 200, "data": []}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"window": "7d", "aggregate": "namespace", "step": "1d"},
			HTTP:          httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "step=1d")
	})
}
