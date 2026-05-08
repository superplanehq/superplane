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

func Test__DeletePool__Setup(t *testing.T) {
	component := &DeletePool{}

	t.Run("missing accountId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "",
				"pool":      "pool123",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "accountId is required")
	})

	t.Run("missing poolId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"pool":      "",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "pool is required")
	})

	t.Run("expression poolId is accepted without API call", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"pool":      "{{ $.trigger.data.poolId }}",
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("accountId from integration metadata is used as fallback", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"pool": "{{ $.trigger.data.poolId }}",
			},
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{AccountID: "acc-from-integration"},
			},
			Metadata: &contexts.MetadataContext{},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("valid configuration resolves pool name", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"pool":      "pool123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"success": true,
							"result": {"id": "pool123", "name": "my-pool"}
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
		poolMeta, ok := meta.Metadata.(PoolNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "my-pool", poolMeta.PoolName)
	})
}

func Test__DeletePool__Execute(t *testing.T) {
	component := &DeletePool{}

	t.Run("successful delete emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"success": true, "result": {"id": "pool123"}}`)),
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
				"pool":      "pool123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, "cloudflare.pool.deleted", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/acc123/load_balancers/pools/pool123", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message": "Pool is in use"}]}`)),
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
				"pool":      "pool123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete pool")
	})
}
