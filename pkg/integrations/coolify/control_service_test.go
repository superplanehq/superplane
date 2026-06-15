package coolify

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

func Test__Coolify_ControlService__Setup(t *testing.T) {
	component := &ControlService{}

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"operation": "start"}})
		require.ErrorContains(t, err, "service is required")
	})

	t.Run("invalid operation -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "svc1", "operation": "purge"},
		})
		require.ErrorContains(t, err, "invalid operation")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"service": "svc1", "operation": "start"},
		})
		require.NoError(t, err)
	})
}

func Test__Coolify_ControlService__Execute(t *testing.T) {
	component := &ControlService{}

	t.Run("start service -> emits payload and calls API", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Service starting."}`)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"service":   "svc1",
				"operation": "start",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, ControlServicePayloadType, executionState.Type)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "svc1", data["serviceUuid"])
		assert.Equal(t, "start", data["operation"])
		assert.Equal(t, "Service starting.", data["message"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://coolify.example.com/api/v1/services/svc1/start", httpCtx.Requests[0].URL.String())
	})
}
