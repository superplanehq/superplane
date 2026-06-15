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

func Test__Coolify_ControlApplication__Setup(t *testing.T) {
	component := &ControlApplication{}

	t.Run("missing application -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"operation": "start"}})
		require.ErrorContains(t, err, "application is required")
	})

	t.Run("missing operation -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: map[string]any{"application": "abc123"}})
		require.ErrorContains(t, err, "operation is required")
	})

	t.Run("invalid operation -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"application": "abc123", "operation": "delete"},
		})
		require.ErrorContains(t, err, "invalid operation")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		for _, op := range []string{"start", "stop", "restart"} {
			err := component.Setup(core.SetupContext{
				Configuration: map[string]any{"application": "abc123", "operation": op},
			})
			require.NoError(t, err, "operation %q should be accepted", op)
		}
	})
}

func Test__Coolify_ControlApplication__Execute(t *testing.T) {
	component := &ControlApplication{}

	t.Run("restart application -> emits payload and calls API", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Application restarting."}`)),
				},
			},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
			Configuration: map[string]any{
				"application": "abc123",
				"operation":   "restart",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, ControlApplicationPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, "abc123", data["applicationUuid"])
		assert.Equal(t, "restart", data["operation"])
		assert.Equal(t, "Application restarting.", data["message"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://coolify.example.com/api/v1/applications/abc123/restart", httpCtx.Requests[0].URL.String())
	})

	t.Run("API error -> wrapped", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Not found"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"application": "missing",
				"operation":   "stop",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "stop application")
		assert.Contains(t, err.Error(), "Not found")
	})
}
