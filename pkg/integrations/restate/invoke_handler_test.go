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

func Test__InvokeHandler__Setup(t *testing.T) {
	component := &InvokeHandler{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"service": "",
				"handler": "myHandler",
			},
		})

		require.ErrorContains(t, err, "service is required")
	})

	t.Run("missing handler -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"service": "MyService",
				"handler": "",
			},
		})

		require.ErrorContains(t, err, "handler is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"service": "MyService",
				"handler": "myHandler",
			},
		})

		require.NoError(t, err)
	})
}

func Test__InvokeHandler__Execute(t *testing.T) {
	component := &InvokeHandler{}

	t.Run("successful invocation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result": "success", "value": 42}`)),
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
				"service": "MyService",
				"handler": "myHandler",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "restate.invocation.response", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "http://localhost:8080/MyService/myHandler", req.URL.String())
	})

	t.Run("invocation with payload and idempotency key", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"ok": true}`)),
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
				"service":        "MyService",
				"handler":        "myHandler",
				"payload":        `{"name": "Mary", "age": 25}`,
				"idempotencyKey": "unique-key-123",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, "unique-key-123", req.Header.Get("Idempotency-Key"))
	})

	t.Run("handler returns error status -> fails execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"code": 500, "message": "handler error"}`)),
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
				"service": "MyService",
				"handler": "myHandler",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err) // Fail is handled via executionState.Fail, not error return
		assert.False(t, executionState.Passed)
		assert.Equal(t, "invocation_error", executionState.FailureReason)
	})

	t.Run("virtual object invocation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"items": []}`)),
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
				"service": "CartObject/user-123",
				"handler": "getCart",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "http://localhost:8080/CartObject/user-123/getCart", httpContext.Requests[0].URL.String())
	})
}

func Test__InvokeHandler__OutputChannels(t *testing.T) {
	component := &InvokeHandler{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}
