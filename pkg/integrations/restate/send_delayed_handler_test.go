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

func Test__SendDelayedHandler__Setup(t *testing.T) {
	component := &SendDelayedHandler{}

	t.Run("missing service -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"service": "",
				"handler": "myHandler",
				"delay":   "30s",
			},
		})

		require.ErrorContains(t, err, "service is required")
	})

	t.Run("missing handler -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"service": "MyService",
				"handler": "",
				"delay":   "30s",
			},
		})

		require.ErrorContains(t, err, "handler is required")
	})

	t.Run("missing delay -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"service": "MyService",
				"handler": "myHandler",
				"delay":   "",
			},
		})

		require.ErrorContains(t, err, "delay is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"service": "MyService",
				"handler": "myHandler",
				"delay":   "5m",
			},
		})

		require.NoError(t, err)
	})
}

func Test__SendDelayedHandler__Execute(t *testing.T) {
	component := &SendDelayedHandler{}

	t.Run("successful delayed send", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader(`{"invocationId": "inv_delayed123", "status": "Accepted"}`)),
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
				"service": "ReminderService",
				"handler": "sendReminder",
				"delay":   "30m",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "restate.invocation.delayed", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Contains(t, req.URL.String(), "/ReminderService/sendReminder/send")
		assert.Contains(t, req.URL.RawQuery, "delay=30m")
	})
}
