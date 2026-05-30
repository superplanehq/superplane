package render

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

func Test__Render_ScaleService__Setup(t *testing.T) {
	component := &ScaleService{}

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"numInstances": 2}})
		require.ErrorContains(t, err, "service is required")
	})

	t.Run("invalid instance count -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"service":      "srv-123",
			"numInstances": 0,
		}})
		require.ErrorContains(t, err, "numInstances must be between 1 and 100")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{
			"service":      "srv-123",
			"numInstances": 3,
		}})
		require.NoError(t, err)
	})
}

func Test__Render_ScaleService__Execute(t *testing.T) {
	component := &ScaleService{}

	t.Run("valid configuration -> scales service and emits payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"service":      "srv-123",
				"numInstances": float64(3),
			},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, ScaleServicePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "srv-123", data["serviceId"])
		assert.Equal(t, 3, data["numInstances"])
		assert.Equal(t, "accepted", data["status"])

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/scale")

		body, err := io.ReadAll(request.Body)
		require.NoError(t, err)

		var requestBody map[string]any
		require.NoError(t, json.Unmarshal(body, &requestBody))
		assert.Equal(t, float64(3), requestBody["numInstances"])
	})
}
