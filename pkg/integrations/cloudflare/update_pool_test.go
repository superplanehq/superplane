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

func Test__UpdatePool__Setup(t *testing.T) {
	component := &UpdatePool{}

	t.Run("missing accountId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "",
				"pool":      "pool123",
			},
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
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "pool is required")
	})

	t.Run("origin missing address returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"pool":      "pool123",
				"origins": []any{
					map[string]any{"name": "o1", "address": "", "enabled": true, "weight": 1},
				},
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "origins[0].address is required")
	})

	t.Run("valid configuration without origins passes", func(t *testing.T) {
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

	t.Run("valid configuration with origins passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"pool":      "pool123",
				"origins": []any{
					map[string]any{"name": "stable", "address": "1.2.3.4", "enabled": true, "weight": 0.9},
					map[string]any{"name": "canary", "address": "1.2.3.5", "enabled": true, "weight": 0.1},
				},
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

func Test__UpdatePool__Execute(t *testing.T) {
	component := &UpdatePool{}

	t.Run("successful update with origins emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": {
							"id": "pool123",
							"name": "my-pool",
							"description": "Test pool",
							"enabled": true,
							"minimum_origins": 1,
							"origins": [
								{"name": "stable", "address": "1.2.3.4", "enabled": true, "weight": 0.9},
								{"name": "canary", "address": "1.2.3.5", "enabled": true, "weight": 0.1}
							]
						}
					}`)),
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
				"origins": []any{
					map[string]any{"name": "stable", "address": "1.2.3.4", "enabled": true, "weight": 0.9},
					map[string]any{"name": "canary", "address": "1.2.3.5", "enabled": true, "weight": 0.1},
				},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, "cloudflare.pool.updated", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/acc123/load_balancers/pools/pool123", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodPatch, httpContext.Requests[0].Method)
	})

	t.Run("successful update without origins emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"success": true,
						"result": {
							"id": "pool123",
							"name": "renamed-pool",
							"enabled": true,
							"minimum_origins": 1,
							"origins": []
						}
					}`)),
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
				"name":      "renamed-pool",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "cloudflare.pool.updated", execState.Type)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message": "Pool not found"}]}`)),
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
		assert.Contains(t, err.Error(), "failed to update pool")
	})
}
