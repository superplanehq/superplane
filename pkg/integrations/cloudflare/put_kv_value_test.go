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

func Test__PutKVValue__Setup(t *testing.T) {
	component := &PutKVValue{}

	t.Run("missing accountId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "",
				"namespace": "ns123",
				"key":       "my-key",
				"value":     "my-value",
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
				"key":       "my-key",
				"value":     "my-value",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "namespace is required")
	})

	t.Run("missing key returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"namespace": "ns123",
				"key":       "",
				"value":     "my-value",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "key is required")
	})

	t.Run("missing value returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"namespace": "ns123",
				"key":       "my-key",
				"value":     "",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "value is required")
	})

	t.Run("accountId from integration metadata is used as fallback", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"namespace": "ns123",
				"key":       "my-key",
				"value":     "my-value",
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
				"key":       "my-key",
				"value":     "my-value",
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
		assert.Equal(t, "my-key", kvMeta.KeyName)
	})
}

func Test__PutKVValue__Execute(t *testing.T) {
	component := &PutKVValue{}

	t.Run("successful put emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": {}}`)),
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
				"key":       "my-key",
				"value":     "my-value",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, "cloudflare.kv.value.put", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/acc123/storage/kv/namespaces/ns123/values/my-key", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
	})

	t.Run("success=false returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message": "write failed"}]}`)),
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
				"key":       "my-key",
				"value":     "my-value",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to put KV value")
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message": "Unauthorized"}]}`)),
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
				"key":       "my-key",
				"value":     "my-value",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to put KV value")
	})
}
