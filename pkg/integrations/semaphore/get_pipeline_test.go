package semaphore

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

func Test__Semaphore_GetPipeline__Setup(t *testing.T) {
	component := &GetPipeline{}

	t.Run("missing pipelineId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{}})
		require.ErrorContains(t, err, "pipelineId is required")
	})

	t.Run("valid pipelineId -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"pipelineId": "ppl-123"}})
		require.NoError(t, err)
	})
}

func Test__Semaphore_GetPipeline__Execute(t *testing.T) {
	component := &GetPipeline{}

	t.Run("valid configuration -> emits pipeline payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"pipeline":{"name":"Initial Pipeline","ppl_id":"ppl-123","wf_id":"wf-456","state":"done","result":"passed"}}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"organizationUrl": "https://example.semaphoreci.com",
					"apiToken":        "token-123",
				},
			},
			ExecutionState: executionState,
			Configuration:  map[string]any{"pipelineId": "ppl-123"},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, GetPipelinePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "Initial Pipeline", data["name"])
		assert.Equal(t, "ppl-123", data["ppl_id"])
		assert.Equal(t, "wf-456", data["wf_id"])
		assert.Equal(t, "done", data["state"])
		assert.Equal(t, "passed", data["result"])

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Contains(t, request.URL.Path, "/api/v1alpha/pipelines/ppl-123")
	})
}

func readMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}

	item, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	return item
}
