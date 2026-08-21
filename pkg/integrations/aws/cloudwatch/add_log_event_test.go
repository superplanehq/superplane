package cloudwatch

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func validAddLogEventConfig() map[string]any {
	return map[string]any{
		"region":    "us-east-1",
		"logGroup":  "/aws/lambda/my-function",
		"logStream": "workflow-run-42",
		"message":   "hello world",
	}
}

func okResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestAddLogEvent_Setup(t *testing.T) {
	component := &AddLogEvent{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: "invalid"})
		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing region -> error", func(t *testing.T) {
		config := validAddLogEventConfig()
		config["region"] = " "
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing log group -> error", func(t *testing.T) {
		config := validAddLogEventConfig()
		config["logGroup"] = " "
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "log group is required")
	})

	t.Run("missing log stream -> error", func(t *testing.T) {
		config := validAddLogEventConfig()
		config["logStream"] = " "
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "log stream is required")
	})

	t.Run("missing message -> error", func(t *testing.T) {
		config := validAddLogEventConfig()
		config["message"] = " "
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "message is required")
	})

	t.Run("invalid timestamp -> error", func(t *testing.T) {
		config := validAddLogEventConfig()
		config["timestamp"] = "not-a-date"
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "invalid timestamp")
	})

	t.Run("datetime-local timestamp from the UI picker -> no error", func(t *testing.T) {
		config := validAddLogEventConfig()
		config["timestamp"] = "2026-05-29T09:00"
		err := component.Setup(core.SetupContext{Configuration: config})
		require.NoError(t, err)
	})

	t.Run("datetime-local timestamp with an explicit timezone -> no error", func(t *testing.T) {
		config := validAddLogEventConfig()
		config["timestamp"] = "2026-05-29T09:00"
		config["timezone"] = "America/New_York"
		err := component.Setup(core.SetupContext{Configuration: config})
		require.NoError(t, err)
	})

	t.Run("invalid timezone -> error", func(t *testing.T) {
		config := validAddLogEventConfig()
		config["timestamp"] = "2026-05-29T09:00"
		config["timezone"] = "not-a-zone"
		err := component.Setup(core.SetupContext{Configuration: config})
		require.ErrorContains(t, err, "invalid timezone")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{Configuration: validAddLogEventConfig()})
		require.NoError(t, err)
	})
}

func TestParseLogEventTimestamp(t *testing.T) {
	t.Run("RFC3339 -> parsed as-is regardless of timezone", func(t *testing.T) {
		parsed, err := parseLogEventTimestamp("2026-05-29T09:00:00Z", "America/New_York")
		require.NoError(t, err)
		assert.Equal(t, "2026-05-29T09:00:00Z", parsed.Format(time.RFC3339))
	})

	t.Run("datetime-local without seconds -> parsed in the given timezone", func(t *testing.T) {
		parsed, err := parseLogEventTimestamp("2026-05-29T09:00", "UTC")
		require.NoError(t, err)
		assert.Equal(t, "2026-05-29T09:00:00Z", parsed.Format(time.RFC3339))
	})

	t.Run("datetime-local with seconds -> parsed in the given timezone", func(t *testing.T) {
		parsed, err := parseLogEventTimestamp("2026-05-29T09:00:30", "UTC")
		require.NoError(t, err)
		assert.Equal(t, "2026-05-29T09:00:30Z", parsed.Format(time.RFC3339))
	})

	t.Run("datetime-local in a non-UTC timezone -> converted to UTC", func(t *testing.T) {
		parsed, err := parseLogEventTimestamp("2026-05-29T09:00", "America/New_York")
		require.NoError(t, err)
		assert.Equal(t, "2026-05-29T13:00:00Z", parsed.Format(time.RFC3339))
	})

	t.Run("invalid timezone -> error", func(t *testing.T) {
		_, err := parseLogEventTimestamp("2026-05-29T09:00", "not-a-zone")
		require.ErrorContains(t, err, "invalid timezone")
	})

	t.Run("garbage -> error", func(t *testing.T) {
		_, err := parseLogEventTimestamp("not-a-date", "UTC")
		require.ErrorContains(t, err, "invalid timestamp")
	})
}

func TestAddLogEvent_Execute(t *testing.T) {
	component := &AddLogEvent{}

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("missing credentials -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration:  validAddLogEventConfig(),
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Integration:    &contexts.IntegrationContext{},
		})

		require.ErrorContains(t, err, "credentials")
	})

	t.Run("valid request -> creates stream and puts log event", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(""),
				okResponse(`{"nextSequenceToken": "token-1"}`),
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		config := validAddLogEventConfig()
		config["timestamp"] = "2026-05-29T09:00:00Z"
		err := component.Execute(core.ExecutionContext{
			Configuration:  config,
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		require.True(t, execState.Passed)
		require.Len(t, execState.Payloads, 1)

		payload := execState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "/aws/lambda/my-function", data["logGroup"])
		assert.Equal(t, "workflow-run-42", data["logStream"])
		assert.Equal(t, "hello world", data["message"])
		assert.Equal(t, "2026-05-29T09:00:00Z", data["timestamp"])

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "Logs_20140328.CreateLogStream", httpContext.Requests[0].Header.Get("X-Amz-Target"))
		assert.Equal(t, "Logs_20140328.PutLogEvents", httpContext.Requests[1].Header.Get("X-Amz-Target"))
	})

	t.Run("log stream already exists -> still puts log event", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"__type": "ResourceAlreadyExistsException", "message": "The specified log stream already exists"}`)),
				},
				okResponse(`{"nextSequenceToken": "token-1"}`),
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration:  validAddLogEventConfig(),
			ExecutionState: execState,
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.NoError(t, err)
		require.True(t, execState.Passed)
		require.Len(t, httpContext.Requests, 2)
	})

	t.Run("create log stream fails with unrelated error -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"__type": "InvalidParameterException", "message": "bad name"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  validAddLogEventConfig(),
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.ErrorContains(t, err, "failed to create log stream")
	})

	t.Run("put log events fails -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(""),
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"__type": "ResourceNotFoundException", "message": "log group not found"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  validAddLogEventConfig(),
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.ErrorContains(t, err, "failed to put log event")
	})

	t.Run("put log events accepted but rejected -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				okResponse(""),
				okResponse(`{"nextSequenceToken": "token-1", "rejectedLogEventsInfo": {"tooOldLogEventEndIndex": 0}}`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  validAddLogEventConfig(),
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{CurrentSecrets: credentialSecrets()},
		})

		require.ErrorContains(t, err, "failed to put log event")
		require.ErrorContains(t, err, "older than 14 days")
	})
}
