package fluxcd

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

func Test__FluxCD__Sync(t *testing.T) {
	integration := &FluxCD{}

	t.Run("successful sync", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"major":"1","minor":"28"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"server":    "https://kubernetes.example.com:6443",
				"token":     "test-token",
				"namespace": "flux-system",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
	})

	t.Run("missing server -> error", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"server": "",
				"token":  "test-token",
			},
		})

		require.ErrorContains(t, err, "server is required")
	})

	t.Run("missing token -> error", func(t *testing.T) {
		err := integration.Sync(core.SyncContext{
			Configuration: map[string]any{
				"server": "https://kubernetes.example.com:6443",
				"token":  "",
			},
		})

		require.ErrorContains(t, err, "token is required")
	})

	t.Run("connection failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Unauthorized"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"server":    "https://kubernetes.example.com:6443",
				"token":     "invalid-token",
				"namespace": "flux-system",
			},
		}

		err := integration.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "failed to connect to Kubernetes API")
	})
}

func Test__FluxCD__Components(t *testing.T) {
	integration := &FluxCD{}
	components := integration.Components()
	require.Len(t, components, 1)
	assert.Equal(t, "fluxcd.reconcileSource", components[0].Name())
}

func Test__FluxCD__Triggers(t *testing.T) {
	integration := &FluxCD{}
	triggers := integration.Triggers()
	require.Len(t, triggers, 1)
	assert.Equal(t, "fluxcd.onReconciliationCompleted", triggers[0].Name())
}
