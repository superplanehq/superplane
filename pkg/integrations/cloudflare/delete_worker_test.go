package cloudflare

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

func Test__DeleteWorker__Setup(t *testing.T) {
	component := &DeleteWorker{}

	t.Run("missing accountId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":  "",
				"scriptName": "w",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "accountId is required")
	})

	t.Run("missing scriptName returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":  "acc",
				"scriptName": "",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "scriptName is required")
	})

	t.Run("valid passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":  "acc",
				"scriptName": "w",
			},
			Integration: &contexts.IntegrationContext{},
		}
		require.NoError(t, component.Setup(ctx))
	})
}

func Test__DeleteWorker__Execute(t *testing.T) {
	component := &DeleteWorker{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"success": true}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiToken": "token"},
	}
	execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}

	ctx := core.ExecutionContext{
		Configuration: map[string]any{
			"accountId":  "acc",
			"scriptName": "gone",
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		ExecutionState: execState,
	}

	require.NoError(t, component.Execute(ctx))
	assert.Equal(t, "cloudflare.worker.deleted", execState.Type)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/accounts/acc/workers/scripts/gone")
}
