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
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"window": "", "aggregate": AggregateNamespace}})
		require.ErrorContains(t, err, "window is required")
	})

	t.Run("aggregate is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"window": WindowOneDay, "aggregate": ""}})
		require.ErrorContains(t, err, "aggregate is required")
	})

	t.Run("invalid window returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"window": "30d", "aggregate": AggregateNamespace}})
		require.ErrorContains(t, err, "invalid window")
	})

	t.Run("invalid aggregate returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"window": WindowOneDay, "aggregate": "pod"}})
		require.ErrorContains(t, err, "invalid aggregate")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"window": WindowOneDay, "aggregate": AggregateNamespace}})
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
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 28.45,
								"gpuCost": 0,
								"ramCost": 18.32,
								"pvCost": 5.67,
								"networkCost": 2.12,
								"totalCost": 54.56
							}
						}]
					}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"window":    WindowOneDay,
				"aggregate": AggregateNamespace,
			},
			HTTP: httpCtx,
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
		assert.Equal(t, "production", payload["name"])
		assert.Equal(t, 54.56, payload["totalCost"])
	})

	t.Run("no data returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"code": 200, "data": [{}]}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"window":    WindowOneDay,
				"aggregate": AggregateNamespace,
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "no allocation data found")
	})

	t.Run("filter returns matching allocations", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"production": {
								"name": "production",
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 28.45,
								"gpuCost": 0,
								"ramCost": 18.32,
								"pvCost": 5.67,
								"networkCost": 2.12,
								"totalCost": 54.56
							},
							"staging": {
								"name": "staging",
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 5.0,
								"gpuCost": 0,
								"ramCost": 3.0,
								"pvCost": 1.0,
								"networkCost": 0.5,
								"totalCost": 9.5
							}
						}]
					}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"window":    WindowOneDay,
				"aggregate": AggregateNamespace,
				"filter":    "production",
			},
			HTTP: httpCtx,
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
		assert.Equal(t, "production", payload["name"])
	})

	t.Run("filter with no match returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"production": {
								"name": "production",
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 28.45,
								"gpuCost": 0,
								"ramCost": 18.32,
								"pvCost": 5.67,
								"networkCost": 2.12,
								"totalCost": 54.56
							}
						}]
					}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"window":    WindowOneDay,
				"aggregate": AggregateNamespace,
				"filter":    "nonexistent",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "no allocation data found for filter")
	})

	t.Run("execute sanitizes configuration", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"data": [{
							"production": {
								"name": "production",
								"start": "2026-02-17T00:00:00Z",
								"end": "2026-02-18T00:00:00Z",
								"cpuCost": 28.45,
								"gpuCost": 0,
								"ramCost": 18.32,
								"pvCost": 5.67,
								"networkCost": 2.12,
								"totalCost": 54.56
							}
						}]
					}`)),
				},
			},
		}

		executionCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"window":    "  1d  ",
				"aggregate": "  NAMESPACE  ",
				"filter":    "  production  ",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "http://opencost.example.com:9003",
				"authType": AuthTypeNone,
			}},
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Passed)
		require.Len(t, executionCtx.Payloads, 1)
	})
}
