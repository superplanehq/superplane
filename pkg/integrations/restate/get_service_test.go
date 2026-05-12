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

func Test__GetService__Setup(t *testing.T) {
	component := &GetService{}

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"service": "",
			},
		})

		require.ErrorContains(t, err, "service is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"service": "CartService",
			},
		})

		require.NoError(t, err)
	})
}

func Test__GetService__Execute(t *testing.T) {
	component := &GetService{}

	t.Run("successful get", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"name": "CartService",
						"revision": 3,
						"ty": "Service",
						"deployment_id": "dp_abc123",
						"public": true
					}`)),
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
			Configuration: map[string]any{
				"service": "CartService",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "restate.service", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/services/CartService")
	})

	t.Run("service not found -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "service not found"}`)),
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
			Configuration: map[string]any{
				"service": "NonExistentService",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get service")
	})
}
