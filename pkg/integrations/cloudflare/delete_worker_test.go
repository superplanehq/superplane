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
				"accountId":    "",
				"workerScript": "w",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "accountId is required")
	})

	t.Run("missing workerScript returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":    "acc",
				"workerScript": "",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "workerScript is required")
	})

	t.Run("expression workerScript skips list API", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":    "acc",
				"workerScript": "{{ $.trigger.data.name }}",
			},
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
		}
		require.NoError(t, component.Setup(ctx))
	})

	t.Run("valid configuration resolves script display name", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId":    "acc123",
				"workerScript": "gone",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"success": true,
							"result": [{"id": "gone", "name": "Gone Worker"}]
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "token123"},
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)

		meta, ok := ctx.Metadata.(*contexts.MetadataContext)
		require.True(t, ok)
		scriptMeta, ok := meta.Metadata.(WorkerScriptNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "Gone Worker", scriptMeta.ScriptDisplayName)
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
			"accountId":    "acc",
			"workerScript": "gone",
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
