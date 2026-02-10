package render

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

func Test__Render_PurgeCache__Setup(t *testing.T) {
	component := &PurgeCache{}

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{}})
		require.ErrorContains(t, err, "service is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"service": "srv-123"}})
		require.NoError(t, err)
	})
}

func Test__Render_PurgeCache__Execute(t *testing.T) {
	component := &PurgeCache{}

	t.Run("valid configuration -> purges cache and emits payload", func(t *testing.T) {
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
			Configuration:  map[string]any{"service": "srv-123"},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, PurgeCachePayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "srv-123", data["serviceId"])
		assert.Equal(t, "accepted", data["status"])

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/cache/purge")
	})
}
