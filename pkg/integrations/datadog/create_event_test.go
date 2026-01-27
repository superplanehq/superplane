package datadog

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

func Test__CreateEvent__Setup(t *testing.T) {
	component := &CreateEvent{}

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})

	t.Run("missing title -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title": "",
				"text":  "Some text",
			},
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("missing text -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title": "Test Event",
				"text":  "",
			},
		})

		require.ErrorContains(t, err, "text is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title": "Test Event",
				"text":  "Event description",
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateEvent__Execute(t *testing.T) {
	component := &CreateEvent{}

	t.Run("successful event creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"event": {
							"id": 12345,
							"title": "Deployment completed",
							"text": "v1.2.3 deployed",
							"date_happened": 1704067200,
							"alert_type": "info",
							"priority": "normal",
							"tags": ["env:prod"],
							"url": "https://app.datadoghq.com/event/event?id=12345"
						},
						"status": "ok"
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"title":     "Deployment completed",
				"text":      "v1.2.3 deployed",
				"alertType": "info",
				"priority":  "normal",
				"tags":      "env:prod",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "datadog.event", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.String(), "/api/v1/events")
		assert.Equal(t, "test-api-key", req.Header.Get("DD-API-KEY"))
		assert.Equal(t, "test-app-key", req.Header.Get("DD-APPLICATION-KEY"))
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"errors": ["Invalid request"]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"title": "Test Event",
				"text":  "Test text",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create event")
	})

	t.Run("event without tags", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"event": {
							"id": 12345,
							"title": "Test Event",
							"text": "Test text",
							"date_happened": 1704067200,
							"alert_type": "info",
							"priority": "normal",
							"tags": [],
							"url": "https://app.datadoghq.com/event/event?id=12345"
						},
						"status": "ok"
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"site":   "datadoghq.com",
				"apiKey": "test-api-key",
				"appKey": "test-app-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"title": "Test Event",
				"text":  "Test text",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
	})
}

func Test__CreateEvent__Configuration(t *testing.T) {
	component := &CreateEvent{}
	config := component.Configuration()

	require.Len(t, config, 5)

	titleField := config[0]
	assert.Equal(t, "title", titleField.Name)
	assert.True(t, titleField.Required)

	textField := config[1]
	assert.Equal(t, "text", textField.Name)
	assert.True(t, textField.Required)

	alertTypeField := config[2]
	assert.Equal(t, "alertType", alertTypeField.Name)
	assert.False(t, alertTypeField.Required)

	priorityField := config[3]
	assert.Equal(t, "priority", priorityField.Name)
	assert.False(t, priorityField.Required)

	tagsField := config[4]
	assert.Equal(t, "tags", tagsField.Name)
	assert.False(t, tagsField.Required)
}

func Test__CreateEvent__OutputChannels(t *testing.T) {
	component := &CreateEvent{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}
