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

func Test__UpdateWorkerRoute__Setup(t *testing.T) {
	component := &UpdateWorkerRoute{}

	t.Run("missing zone returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "",
				"pattern": "ex.com/*",
				"script":  "w",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "zone is required")
	})

	t.Run("missing pattern returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "z1",
				"pattern": "",
				"script":  "w",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "pattern is required")
	})

	t.Run("missing script returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "z1",
				"pattern": "ex.com/*",
				"script":  "",
			},
		}
		require.ErrorContains(t, component.Setup(ctx), "script is required")
	})

	t.Run("valid create shape passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":    "z1",
				"pattern": "ex.com/*",
				"script":  "w",
			},
		}
		require.NoError(t, component.Setup(ctx))
	})
}

func Test__UpdateWorkerRoute__Execute__create(t *testing.T) {
	component := &UpdateWorkerRoute{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"success": true,
					"result": { "id": "route-1", "pattern": "ex.com/*", "script": "w" }
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
			"zone":    "zone-id",
			"pattern": "ex.com/*",
			"script":  "w",
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		ExecutionState: execState,
	}

	require.NoError(t, component.Execute(ctx))
	assert.Equal(t, "cloudflare.workerRoute.created", execState.Type)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/zones/zone-id/workers/routes")
}

func Test__UpdateWorkerRoute__Execute__update(t *testing.T) {
	component := &UpdateWorkerRoute{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"success": true,
					"result": { "id": "route-1", "pattern": "ex.com/api/*", "script": "w2" }
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
			"zone":    "zone-id",
			"routeId": "route-1",
			"pattern": "ex.com/api/*",
			"script":  "w2",
		},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		ExecutionState: execState,
	}

	require.NoError(t, component.Execute(ctx))
	assert.Equal(t, "cloudflare.workerRoute.updated", execState.Type)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/zones/zone-id/workers/routes/route-1")
}
