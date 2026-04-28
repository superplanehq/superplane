package restate

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

func Test__HealthCheck__Execute(t *testing.T) {
	component := &HealthCheck{}

	t.Run("healthy cluster", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// health check
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
				// cluster health
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"nodes": {"0": {"status": "ALIVE"}}}`)),
				},
				// version
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"version": "1.1.4"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"adminUrl":   "http://localhost:9070",
				"ingressUrl": "http://localhost:8080",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "restate.health", executionState.Type)

		// Should have made 3 requests: health, cluster-health, version
		require.Len(t, httpContext.Requests, 3)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/health")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/cluster-health")
		assert.Contains(t, httpContext.Requests[2].URL.String(), "/version")
	})

	t.Run("unhealthy cluster -> fails execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusServiceUnavailable,
					Body:       io.NopCloser(strings.NewReader(`{"message": "not ready"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"adminUrl":   "http://localhost:9070",
				"ingressUrl": "http://localhost:8080",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err) // Fail is handled via executionState.Fail
		assert.False(t, executionState.Passed)
		assert.Equal(t, "unhealthy", executionState.FailureReason)
	})
}
