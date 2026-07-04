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

func Test__DeleteKVNamespace__Setup(t *testing.T) {
	component := &DeleteKVNamespace{}

	t.Run("missing accountId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "",
				"namespace": "ns123",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "accountId is required")
	})

	t.Run("missing namespaceId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"namespace": "",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "namespace is required")
	})

	t.Run("accountId from integration metadata is used as fallback", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"namespace": "ns123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":{"id":"ns123","title":"My Namespace"}}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "token123"},
				Metadata:      Metadata{AccountID: "acc-from-integration"},
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("valid configuration resolves namespace name", func(t *testing.T) {
		metaCtx := &contexts.MetadataContext{}
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"namespace": "ns123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"success":true,"result":{"id":"ns123","title":"My Namespace"}}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "token123"},
			},
			Metadata: metaCtx,
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
		kvMeta, ok := metaCtx.Metadata.(KVNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "My Namespace", kvMeta.NamespaceName)
	})
}

func Test__DeleteKVNamespace__Execute(t *testing.T) {
	component := &DeleteKVNamespace{}

	t.Run("successful delete emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": null}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"namespace": "ns123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, "cloudflare.kv.namespace.deleted", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/acc123/storage/kv/namespaces/ns123", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message": "namespace not found"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"namespace": "ns123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete KV namespace")
	})
}
