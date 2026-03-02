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

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing labels -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"labels":  "",
				"message": "hello",
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

	t.Run("invalid label format -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"labels":  "invalid-label",
				"message": "hello",
			},
		})

		require.ErrorContains(t, err, "invalid labels")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"labels":  "job=superplane,env=prod",
				"message": "test message",
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
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"labels":  "job=superplane,env=prod",
				"message": "Deployment completed",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "loki.pushLogs", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.String(), "/loki/api/v1/push")
	})

	t.Run("push failure -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"message":"invalid stream"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"labels":  "job=test",
				"message": "hello",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to push logs")
	})

	t.Run("invalid labels -> returns error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://loki.example.com",
				"authType": AuthTypeNone,
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"labels":  "invalid",
				"message": "hello",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    appCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid labels")
	})
}

func Test__PushLogs__OutputChannels(t *testing.T) {
	component := &PushLogs{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func Test__PushLogs__parseLabels(t *testing.T) {
	t.Run("single label", func(t *testing.T) {
		labels, err := parseLabels("job=superplane")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"job": "superplane"}, labels)
	})

	t.Run("multiple labels", func(t *testing.T) {
		labels, err := parseLabels("job=superplane,env=prod")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"job": "superplane", "env": "prod"}, labels)
	})

	t.Run("labels with spaces", func(t *testing.T) {
		labels, err := parseLabels("job = superplane , env = prod")
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"job": "superplane", "env": "prod"}, labels)
	})

	t.Run("empty string -> error", func(t *testing.T) {
		_, err := parseLabels("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one label is required")
	})

	t.Run("invalid format -> error", func(t *testing.T) {
		_, err := parseLabels("noequalssign")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid label format")
	})

	t.Run("empty key -> error", func(t *testing.T) {
		_, err := parseLabels("=value")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "label key cannot be empty")
	})
}
