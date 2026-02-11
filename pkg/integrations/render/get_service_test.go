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

func Test__Render_GetService__Setup(t *testing.T) {
	component := &GetService{}

	t.Run("missing serviceId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})
		require.ErrorContains(t, err, "serviceId is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"serviceId": "srv-abc123"},
		})
		require.NoError(t, err)
	})
}

func Test__Render_GetService__Execute(t *testing.T) {
	component := &GetService{}

	t.Run("valid input -> fetches service and emits", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"srv-abc123","name":"my-service","type":"web_service","suspended":"not_suspended","createdAt":"2026-01-01T00:00:00Z","updatedAt":"2026-02-01T00:00:00Z"}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"serviceId": "srv-abc123"},
		})

		require.NoError(t, err)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, GetServicePayloadType, executionState.Type)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/services/srv-abc123")
	})

	t.Run("missing serviceId -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Configuration:  map[string]any{},
		})
		require.ErrorContains(t, err, "serviceId is required")
	})
}
