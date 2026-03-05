package dash0

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

func Test__SendLogEvent__Setup(t *testing.T) {
	component := SendLogEvent{}

	t.Run("body is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"body": ""},
		})

		require.ErrorContains(t, err, "body is required")
	})

	t.Run("body cannot be empty", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"body": "   "},
		})

		require.ErrorContains(t, err, "body cannot be empty")
	})

	t.Run("valid setup with minimal config", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"body": "Test log message"},
		})

		require.NoError(t, err)
	})

	t.Run("valid setup with all fields", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"body":         "Test log message",
				"severityText": "ERROR",
				"dataset":      "production",
				"attributes": map[string]string{
					"service": "api",
					"version": "1.0.0",
				},
			},
		})

		require.NoError(t, err)
	})
}

func Test__SendLogEvent__Execute(t *testing.T) {
	component := SendLogEvent{}

	t.Run("successful log send with minimal config", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"sent": true}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"body": "Deployment completed",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "dash0.log.sent", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("successful log send with all fields", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"sent": true}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		severityText := "ERROR"
		dataset := "production"
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"body":         "Deployment failed",
				"severityText": severityText,
				"dataset":      dataset,
				"attributes": map[string]string{
					"service": "api",
					"version": "1.0.0",
				},
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "dash0.log.sent", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("successful log send with empty response body", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"body": "Test log",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
	})

	t.Run("log send failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":"invalid payload"}`)),
				},
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"body": "Test log",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send log event")
	})

	t.Run("missing integration config -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		execCtx := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"body": "Test log",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{},
			},
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error creating client")
	})
}

func Test__SendLogEvent__getSeverityNumber(t *testing.T) {
	component := SendLogEvent{}

	tests := []struct {
		severity string
		expected int
	}{
		{"TRACE", 1},
		{"DEBUG", 5},
		{"INFO", 9},
		{"WARN", 13},
		{"ERROR", 17},
		{"FATAL", 21},
		{"trace", 1}, // Test case-insensitivity
		{"info", 9},
		{"unknown", 9}, // Unknown defaults to INFO
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			result := component.getSeverityNumber(&tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("nil severity -> defaults to INFO", func(t *testing.T) {
		result := component.getSeverityNumber(nil)
		assert.Equal(t, 9, result)
	})
}
