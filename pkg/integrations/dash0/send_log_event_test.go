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

		// Verify the payload includes log record details
		wrapped, ok := execCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, payload["sent"])
		assert.Equal(t, "INFO", payload["severityText"])
		assert.Equal(t, "Deployment completed", payload["body"])
		assert.Equal(t, "default", payload["dataset"])

		// Verify the request was sent to the correct OTLP ingress endpoint
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, "https://ingress.us-west-2.aws.dash0.com:4318/v1/logs", req.URL.String())
		assert.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		// Default dataset should not set Dash0-Dataset header
		assert.Empty(t, req.Header.Get("Dash0-Dataset"))
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

		// Verify the payload includes log record details
		wrapped, ok := execCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		payload, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, payload["sent"])
		assert.Equal(t, "ERROR", payload["severityText"])
		assert.Equal(t, "Deployment failed", payload["body"])
		assert.Equal(t, "production", payload["dataset"])
		attribs, ok := payload["attributes"].(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "api", attribs["service"])
		assert.Equal(t, "1.0.0", attribs["version"])

		// Verify the request includes the Dash0-Dataset header for non-default dataset
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, "https://ingress.us-west-2.aws.dash0.com:4318/v1/logs", req.URL.String())
		assert.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
		assert.Equal(t, "production", req.Header.Get("Dash0-Dataset"))
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

func Test__severityTextToNumber(t *testing.T) {
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
			result := severityTextToNumber(tt.severity)
			assert.Equal(t, tt.expected, result)
		})
	}

	t.Run("empty string -> defaults to INFO", func(t *testing.T) {
		result := severityTextToNumber("")
		assert.Equal(t, 9, result)
	})
}

func Test__deriveOTLPEndpoint(t *testing.T) {
	t.Run("derives ingress endpoint from API base URL", func(t *testing.T) {
		result, err := deriveOTLPEndpoint("https://api.us-west-2.aws.dash0.com")
		require.NoError(t, err)
		assert.Equal(t, "https://ingress.us-west-2.aws.dash0.com:4318", result)
	})

	t.Run("derives ingress endpoint from EU region", func(t *testing.T) {
		result, err := deriveOTLPEndpoint("https://api.eu-west-1.aws.dash0.com")
		require.NoError(t, err)
		assert.Equal(t, "https://ingress.eu-west-1.aws.dash0.com:4318", result)
	})

	t.Run("handles trailing slash in API base URL", func(t *testing.T) {
		result, err := deriveOTLPEndpoint("https://api.us-west-2.aws.dash0.com/")
		require.NoError(t, err)
		assert.Equal(t, "https://ingress.us-west-2.aws.dash0.com:4318", result)
	})

	t.Run("returns error for non-api hostname", func(t *testing.T) {
		_, err := deriveOTLPEndpoint("https://custom.dash0.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not start with 'api.'")
	})

	t.Run("returns error for empty URL", func(t *testing.T) {
		_, err := deriveOTLPEndpoint("")
		require.Error(t, err)
	})
}
