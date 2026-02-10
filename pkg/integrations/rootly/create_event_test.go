package rootly

import (
	"encoding/json"
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

	t.Run("missing incidentId -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"event": "Update",
			},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("missing event -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
			},
		})

		require.ErrorContains(t, err, "event is required")
	})

	t.Run("invalid visibility -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
				"event":      "Update",
				"visibility": "public",
			},
		})

		require.ErrorContains(t, err, "visibility must be internal or external")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
				"event":      "Update",
				"visibility": "internal",
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
						"data": {
							"id": "evt-123",
							"type": "incident_events",
							"attributes": {
								"event": "Investigation update",
								"visibility": "internal",
								"occurred_at": "2026-01-19T12:15:00Z",
								"created_at": "2026-01-19T12:15:00Z"
							}
						}
					}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
				"event":      "Investigation update",
				"visibility": "internal",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "rootly.incident.event", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://api.rootly.com/v1/incidents/inc-123/events", req.URL.String())
		assert.Equal(t, "application/vnd.api+json", req.Header.Get("Content-Type"))
		assert.Equal(t, "application/vnd.api+json", req.Header.Get("Accept"))
		assert.Equal(t, "Bearer test-api-key", req.Header.Get("Authorization"))

		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)

		var payload map[string]any
		err = json.Unmarshal(body, &payload)
		require.NoError(t, err)

		data := payload["data"].(map[string]any)
		assert.Equal(t, "incident_events", data["type"])
		attributes := data["attributes"].(map[string]any)
		assert.Equal(t, "Investigation update", attributes["event"])
		assert.Equal(t, "internal", attributes["visibility"])
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error":"invalid"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
				"event":      "Investigation update",
			},
			HTTP:           httpContext,
			Integration:    appCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create incident event")
	})
}

func Test__CreateEvent__OutputChannels(t *testing.T) {
	component := &CreateEvent{}
	channels := component.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}
