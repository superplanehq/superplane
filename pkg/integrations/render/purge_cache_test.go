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

func Test__Render_PurgeCache__Execute(t *testing.T) {
	component := &PurgeCache{}

	t.Run("valid input -> purges cache and emits", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
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
		assert.Equal(t, PurgeCachePayloadType, executionState.Type)

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPost, httpCtx.Requests[0].Method)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/v1/services/srv-abc123/cache/purge")
	})
}
