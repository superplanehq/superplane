package loki

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

func Test__PushLogs__Setup(t *testing.T) {
	component := &PushLogs{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})

	t.Run("missing labels -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"labels":  "",
				"message": "test message",
			},
		})

		require.ErrorContains(t, err, "labels is required")
	})

	t.Run("missing message -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"labels":  "job=test",
				"message": "",
			},
		})

		require.ErrorContains(t, err, "message is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"labels":  "job=superplane,env=prod",
				"message": "Test log message",
			},
		})

		require.NoError(t, err)
	})
}

func Test__PushLogs__Execute(t *testing.T) {
	component := &PushLogs{}

	t.Run("successful push", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "http://loki:3100",
				"authType": "none",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"labels":  "job=superplane,env=prod",
				"message": "Deployment completed",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "loki.logEntry", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.String(), "/loki/api/v1/push")
	})

	t.Run("push with basic auth", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "http://loki:3100",
				"authType": "basic",
				"username": "admin",
				"password": "secret",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"labels":  "job=test",
				"message": "Test message",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Contains(t, req.Header.Get("Authorization"), "Basic ")
	})

	t.Run("push with bearer token", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "http://loki:3100",
				"authType": "bearer",
				"token":    "my-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"labels":  "job=test",
				"message": "Test message",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, "Bearer my-token", req.Header.Get("Authorization"))
	})

	t.Run("push with tenant ID", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "http://loki:3100",
				"authType": "none",
				"tenantId": "my-tenant",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"labels":  "job=test",
				"message": "Test message",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, "my-tenant", req.Header.Get("X-Scope-OrgID"))
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error": "bad request"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"url":      "http://loki:3100",
				"authType": "none",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"labels":  "job=test",
				"message": "Test message",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to push logs")
	})
}

func Test__PushLogs__OutputChannels(t *testing.T) {
	component := &PushLogs{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func Test__ParseLabels(t *testing.T) {
	t.Run("single label", func(t *testing.T) {
		result := parseLabels("job=superplane")
		assert.Equal(t, map[string]string{"job": "superplane"}, result)
	})

	t.Run("multiple labels", func(t *testing.T) {
		result := parseLabels("job=superplane,env=prod")
		assert.Equal(t, map[string]string{"job": "superplane", "env": "prod"}, result)
	})

	t.Run("labels with spaces", func(t *testing.T) {
		result := parseLabels(" job = superplane , env = prod ")
		assert.Equal(t, map[string]string{"job": "superplane", "env": "prod"}, result)
	})

	t.Run("empty string", func(t *testing.T) {
		result := parseLabels("")
		assert.Equal(t, map[string]string{}, result)
	})

	t.Run("label without value", func(t *testing.T) {
		result := parseLabels("job")
		assert.Equal(t, map[string]string{}, result)
	})
}
