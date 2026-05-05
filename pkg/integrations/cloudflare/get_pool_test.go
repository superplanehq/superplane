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

func Test__GetPool__Setup(t *testing.T) {
	component := &GetPool{}

	t.Run("missing accountId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "",
				"poolId":    "pool123",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "accountId is required")
	})

	t.Run("missing poolId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"poolId":    "",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "poolId is required")
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"accountId": "acc123",
				"poolId":    "pool123",
			},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__GetPool__Execute(t *testing.T) {
	component := &GetPool{}

	t.Run("successful get emits result", func(t *testing.T) {
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
								{"name": "o1", "address": "1.2.3.4", "enabled": true, "weight": 1}
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
				"poolId":    "pool123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, "cloudflare.pool", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/acc123/load_balancers/pools/pool123", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
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
				"poolId":    "pool123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		}

		err := component.Execute(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get pool")
	})
}
