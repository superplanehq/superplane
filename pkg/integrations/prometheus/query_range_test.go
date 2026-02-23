package prometheus

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

func Test__QueryRange__Setup(t *testing.T) {
	component := &QueryRange{}

	t.Run("query is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": "",
				"start": "2026-01-01T00:00:00Z",
				"end":   "2026-01-02T00:00:00Z",
				"step":  "15s",
			},
		})
		require.ErrorContains(t, err, "query is required")
	})

	t.Run("start is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": "up",
				"start": "",
				"end":   "2026-01-02T00:00:00Z",
				"step":  "15s",
			},
		})
		require.ErrorContains(t, err, "start is required")
	})

	t.Run("end is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": "up",
				"start": "2026-01-01T00:00:00Z",
				"end":   "",
				"step":  "15s",
			},
		})
		require.ErrorContains(t, err, "end is required")
	})

	t.Run("step is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": "up",
				"start": "2026-01-01T00:00:00Z",
				"end":   "2026-01-02T00:00:00Z",
				"step":  "",
			},
		})
		require.ErrorContains(t, err, "step is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"query": "up",
				"start": "2026-01-01T00:00:00Z",
				"end":   "2026-01-02T00:00:00Z",
				"step":  "15s",
			},
		})
		require.NoError(t, err)
	})
}

func Test__QueryRange__Execute(t *testing.T) {
	component := &QueryRange{}

	queryRangeResponseJSON := `{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"__name__":"up","job":"prometheus"},"values":[[1708000000,"1"],[1708000015,"1"]]}]}}`

	t.Run("query range result is emitted", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(queryRangeResponseJSON)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		executionCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query": "up",
				"start": "2026-01-01T00:00:00Z",
				"end":   "2026-01-02T00:00:00Z",
				"step":  "15s",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			Metadata:       metadataCtx,
			ExecutionState: executionCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionCtx.Finished)
		assert.True(t, executionCtx.Passed)
		assert.Equal(t, "prometheus.queryRange", executionCtx.Type)
		require.Len(t, executionCtx.Payloads, 1)

		wrappedPayload := executionCtx.Payloads[0].(map[string]any)
		payload := wrappedPayload["data"].(map[string]any)
		assert.Equal(t, "matrix", payload["resultType"])
		assert.NotNil(t, payload["result"])

		metadata := metadataCtx.Metadata.(QueryRangeNodeMetadata)
		assert.Equal(t, "up", metadata.Query)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"status":"error","errorType":"bad_data","error":"invalid query"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"query": "invalid{",
				"start": "2026-01-01T00:00:00Z",
				"end":   "2026-01-02T00:00:00Z",
				"step":  "15s",
			},
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{
				"baseURL":  "https://prometheus.example.com",
				"authType": AuthTypeNone,
			}},
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to execute query range")
	})
}
