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

func Test__GetWorker__Setup(t *testing.T) {
	component := &GetWorker{}

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

func Test__GetWorker__Execute(t *testing.T) {
	component := &GetWorker{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"success": true,
					"result": { "compatibility_date": "2024-01-01", "bindings": [] }
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"success": true,
					"result": { "deployments": [{"id": "d1"}] }
				}`)),
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
			"scriptName": "my-worker",
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		ExecutionState: execState,
	}

	require.NoError(t, component.Execute(ctx))
	assert.Equal(t, "cloudflare.worker.metadata", execState.Type)
	require.Len(t, httpContext.Requests, 2)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/settings")
	assert.Contains(t, httpContext.Requests[1].URL.String(), "/deployments")
}
