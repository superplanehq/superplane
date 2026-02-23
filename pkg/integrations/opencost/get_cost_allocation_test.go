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

	t.Run("missing window -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"aggregate": "namespace",
			},
		})
		require.ErrorContains(t, err, "window is required")
	})

	t.Run("missing aggregate -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"window": "1d",
			},
		})
		require.ErrorContains(t, err, "aggregate is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"window":    "1d",
				"aggregate": "namespace",
			},
		})
		require.NoError(t, err)
	})

	t.Run("valid configuration with step -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"window":    "7d",
				"aggregate": "cluster",
				"step":      "1d",
			},
		})
		require.NoError(t, err)
	})
}

func Test__GetCostAllocation__Execute(t *testing.T) {
	component := &GetCostAllocation{}

	t.Run("valid configuration -> emits allocation payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"code": 200,
						"status": "success",
						"data": [{
							"kube-system": {
								"name": "kube-system",
								"start": "2026-02-22T00:00:00Z",
								"end": "2026-02-23T00:00:00Z",
								"cpuCost": 1.25,
								"gpuCost": 0,
								"ramCost": 0.75,
								"pvCost": 0.1,
								"networkCost": 0.05,
								"sharedCost": 0,
								"externalCost": 0,
								"totalCost": 2.15,
								"totalEfficiency": 0.45,
								"cpuEfficiency": 0.5,
								"ramEfficiency": 0.4
							}
						}]
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiURL": "http://opencost:9003"}},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"window":    "1d",
				"aggregate": "namespace",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, GetCostAllocationPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "1d", data["window"])
		assert.Equal(t, "namespace", data["aggregate"])

		items, ok := data["items"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, items, 1)
		assert.Equal(t, "kube-system", items[0]["name"])
		assert.Equal(t, 2.15, items[0]["totalCost"])

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Contains(t, request.URL.String(), "/allocation")
		assert.Contains(t, request.URL.RawQuery, "window=1d")
		assert.Contains(t, request.URL.RawQuery, "aggregate=namespace")
	})

	t.Run("with step parameter -> includes step in query", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"code":200,"status":"success","data":[]}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiURL": "http://opencost:9003"}},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"window":    "7d",
				"aggregate": "cluster",
				"step":      "1d",
			},
		})

		require.NoError(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.RawQuery, "step=1d")
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error":"internal error"}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiURL": "http://opencost:9003"}},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"window":    "1d",
				"aggregate": "namespace",
			},
		})

		require.ErrorContains(t, err, "failed to fetch cost allocation")
	})
}

func readMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}
