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
				"workerScript": "w1",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"success": true,
							"result": [{"id": "w1", "name": "My Worker"}]
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
		assert.Equal(t, "My Worker", scriptMeta.ScriptDisplayName)
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
			"accountId":    "acc",
			"workerScript": "my-worker",
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		ExecutionState: execState,
	}

	require.NoError(t, component.Execute(ctx))
	assert.Equal(t, "cloudflare.worker.fetched", execState.Type)
	require.Len(t, httpContext.Requests, 2)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/settings")
	assert.Contains(t, httpContext.Requests[1].URL.String(), "/deployments")
}
