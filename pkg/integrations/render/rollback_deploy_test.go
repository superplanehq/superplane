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

func Test__Render_RollbackDeploy__Setup(t *testing.T) {
	component := &RollbackDeploy{}

	t.Run("missing serviceId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"deployId": "dep-abc123"},
		})
		require.ErrorContains(t, err, "serviceId is required")
	})

	t.Run("missing deployId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"serviceId": "srv-abc123"},
		})
		require.ErrorContains(t, err, "deployId is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{"serviceId": "srv-abc123", "deployId": "dep-abc123"},
		})
		require.NoError(t, err)
	})
}

func Test__Render_RollbackDeploy__Execute(t *testing.T) {
	component := &RollbackDeploy{}

	t.Run("valid input -> triggers rollback and emits", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"id":"dep-new123","status":"build_in_progress","createdAt":"2026-02-01T00:00:00Z","finishedAt":""}`,
					)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "rnd_test"}},
			ExecutionState: executionState,
			Configuration:  map[string]any{"serviceId": "srv-abc123", "deployId": "dep-abc123"},
		})

		require.NoError(t, err)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, RollbackDeployPayloadType, executionState.Type)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/services/srv-abc123/rollbacks")
	})
}
