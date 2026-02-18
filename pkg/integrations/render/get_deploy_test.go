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

func Test__Render_GetDeploy__Setup(t *testing.T) {
	component := &GetDeploy{}

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"deployId": "dep-123"}})
		require.ErrorContains(t, err, "service is required")
	})

	t.Run("missing deployId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"service": "srv-123"}})
		require.ErrorContains(t, err, "deployId is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"service": "srv-123", "deployId": "dep-123"}})
		require.NoError(t, err)
	})
}

func Test__Render_GetDeploy__Execute(t *testing.T) {
	component := &GetDeploy{}

	t.Run("valid configuration -> emits deploy payload", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"dep-123","status":"live","trigger":"api","createdAt":"2026-02-05T16:10:00.000000Z"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"service": "srv-123", "deployId": "dep-123"},
		})

		require.NoError(t, err)

		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, GetDeployPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		emittedPayload := readMap(executionState.Payloads[0])
		data := readMap(emittedPayload["data"])
		assert.Equal(t, "srv-123", data["serviceId"])
		assert.Equal(t, "dep-123", data["deployId"])
		assert.Equal(t, "live", data["status"])
		assert.Equal(t, "api", data["trigger"])

		require.Len(t, httpCtx.Requests, 1)
		request := httpCtx.Requests[0]
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Contains(t, request.URL.Path, "/v1/services/srv-123/deploys/dep-123")
	})
}
