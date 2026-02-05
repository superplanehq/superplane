package rootly

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

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
				"event":      "Investigation started",
				"visibility": "internal",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing incident ID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"event":      "Investigation started",
				"visibility": "internal",
			},
		})

		require.ErrorContains(t, err, "incident ID is required")
	})

	t.Run("empty incident ID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "",
				"event":      "Investigation started",
				"visibility": "internal",
			},
		})

		require.ErrorContains(t, err, "incident ID is required")
	})

	t.Run("missing event returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
				"visibility": "internal",
			},
		})

		require.ErrorContains(t, err, "event is required")
	})

	t.Run("empty event returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
				"event":      "",
				"visibility": "internal",
			},
		})

		require.ErrorContains(t, err, "event is required")
	})

	t.Run("visibility is optional", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
				"event":      "Investigation started",
			},
		})

		require.NoError(t, err)
	})

	t.Run("invalid configuration format -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})
}

func Test__CreateEvent__Execute(t *testing.T) {
	component := &CreateEvent{}

	t.Run("successful event creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`
						{
							"data": {
								"id": "evt-456",
								"type": "incident_events",
								"attributes": {
									"event": "Investigation started",
									"visibility": "internal",
									"occurred_at": "2024-01-15T10:30:00.000Z",
									"created_at": "2024-01-15T10:30:00.000Z",
									"updated_at": "2024-01-15T10:30:00.000Z"
								}
							}
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
				"event":      "Investigation started",
				"visibility": "internal",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.rootly.com/v1/incidents/inc-123/events", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)

		// Verify emission
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "rootly.incidentEvent", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
	})

	t.Run("event creation without visibility", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`
						{
							"data": {
								"id": "evt-789",
								"type": "incident_events",
								"attributes": {
									"event": "Status update",
									"visibility": "external",
									"occurred_at": "2024-01-15T11:00:00.000Z",
									"created_at": "2024-01-15T11:00:00.000Z",
									"updated_at": "2024-01-15T11:00:00.000Z"
								}
							}
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "inc-123",
				"event":      "Status update",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Incident not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "invalid-id",
				"event":      "Test event",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create incident event")
	})
}
