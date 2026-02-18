package statuspage

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

const sampleIncidentJSON = `{
	"id": "p31zjtct2jer",
	"name": "Database Connection Issues",
	"status": "investigating",
	"impact": "major",
	"shortlink": "http://stspg.io/p31zjtct2jer",
	"created_at": "2026-02-12T10:30:00.000Z",
	"updated_at": "2026-02-12T10:30:00.000Z",
	"page_id": "kctbh9vrtdwd",
	"components": [],
	"incident_updates": []
}`

func Test__CreateIncident__Setup(t *testing.T) {
	component := &CreateIncident{}

	t.Run("valid configuration realtime", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":         "kctbh9vrtdwd",
				"incidentType": "realtime",
				"name":         "Test Incident",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("valid configuration scheduled", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incidentType":   "scheduled",
				"name":           "Scheduled Maintenance",
				"scheduledFor":   "2026-02-15T02:00:00Z",
				"scheduledUntil": "2026-02-15T04:00:00Z",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("scheduled missing scheduledFor returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incidentType":   "scheduled",
				"name":           "Scheduled Maintenance",
				"scheduledUntil": "2026-02-15T04:00:00Z",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "scheduledFor is required for scheduled incidents")
	})

	t.Run("scheduled missing scheduledUntil returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":         "kctbh9vrtdwd",
				"incidentType": "scheduled",
				"name":         "Scheduled Maintenance",
				"scheduledFor": "2026-02-15T02:00:00Z",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "scheduledUntil is required for scheduled incidents")
	})

	t.Run("missing page returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentType": "realtime",
				"name":         "Test",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page is required")
	})

	t.Run("page not found returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"other-page","name":"Other"}]`))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":         "nonexistent-page",
				"incidentType": "realtime",
				"name":         "Test Incident",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page \"nonexistent-page\" not found or not accessible")
	})

	t.Run("missing incidentType returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page": "kctbh9vrtdwd",
				"name": "Test",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incidentType is required")
	})

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":         "kctbh9vrtdwd",
				"incidentType": "realtime",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("invalid incidentType returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":         "kctbh9vrtdwd",
				"incidentType": "invalid",
				"name":         "Test",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incidentType must be realtime or scheduled")
	})

	t.Run("scheduled with scheduledFor after scheduledUntil returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incidentType":   "scheduled",
				"name":           "Scheduled Maintenance",
				"scheduledFor":   "2026-02-15T06:00",
				"scheduledUntil": "2026-02-15T04:00",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "scheduledFor must be before scheduledUntil")
	})

	t.Run("scheduled with equal scheduledFor and scheduledUntil returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incidentType":   "scheduled",
				"name":           "Scheduled Maintenance",
				"scheduledFor":   "2026-02-15T04:00",
				"scheduledUntil": "2026-02-15T04:00",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "scheduledFor must be before scheduledUntil")
	})

	t.Run("scheduled with invalid scheduledFor returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incidentType":   "scheduled",
				"name":           "Scheduled Maintenance",
				"scheduledFor":   "not-a-valid-datetime",
				"scheduledUntil": "2026-02-15T04:00:00Z",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid scheduledFor")
	})

	t.Run("scheduled with invalid scheduledUntil returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incidentType":   "scheduled",
				"name":           "Scheduled Maintenance",
				"scheduledFor":   "2026-02-15T02:00:00Z",
				"scheduledUntil": "invalid-datetime",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid scheduledUntil")
	})

	t.Run("scheduled with invalid timezone returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":              "kctbh9vrtdwd",
				"incidentType":      "scheduled",
				"name":              "Scheduled Maintenance",
				"scheduledFor":      "2026-02-15T02:00",
				"scheduledUntil":    "2026-02-15T04:00",
				"scheduledTimezone": "Invalid/Timezone",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid scheduledFor")
	})

	t.Run("valid configuration with components", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":         "kctbh9vrtdwd",
				"incidentType": "realtime",
				"name":         "Test Incident",
				"components": []any{
					map[string]any{"componentId": "comp1", "status": "degraded_performance"},
					map[string]any{"componentId": "comp2", "status": "operational"},
				},
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__CreateIncident__Execute(t *testing.T) {
	component := &CreateIncident{}

	t.Run("success realtime emits incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(sampleIncidentJSON)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incidentType":   "realtime",
				"name":           "Database Connection Issues",
				"body":           "We are investigating.",
				"statusRealtime": "investigating",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "statuspage.incident", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "p31zjtct2jer", data["id"])
		assert.Equal(t, "Database Connection Issues", data["name"])
		assert.Equal(t, "investigating", data["status"])
		assert.Equal(t, "major", data["impact"])
		assert.Equal(t, "http://stspg.io/p31zjtct2jer", data["shortlink"])
	})

	t.Run("success scheduled emits incident", func(t *testing.T) {
		scheduledJSON := `{
			"id": "sched123",
			"name": "Maintenance",
			"status": "scheduled",
			"impact": "none",
			"shortlink": "http://stspg.io/sched123",
			"page_id": "kctbh9vrtdwd",
			"scheduled_for": "2026-02-15T02:00:00Z",
			"scheduled_until": "2026-02-15T04:00:00Z",
			"components": [],
			"incident_updates": []
		}`
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(scheduledJSON)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"page":            "kctbh9vrtdwd",
				"incidentType":    "scheduled",
				"name":            "Maintenance",
				"scheduledFor":    "2026-02-15T02:00:00Z",
				"scheduledUntil":  "2026-02-15T04:00:00Z",
				"statusScheduled": "scheduled",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.Equal(t, "statuspage.incident", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "sched123", data["id"])
		assert.Equal(t, "scheduled", data["status"])
	})

	t.Run("success scheduled with statusScheduled in_progress", func(t *testing.T) {
		scheduledInProgressJSON := `{
			"id": "sched456",
			"name": "Maintenance In Progress",
			"status": "in_progress",
			"impact": "none",
			"shortlink": "http://stspg.io/sched456",
			"page_id": "kctbh9vrtdwd",
			"scheduled_for": "2026-02-15T02:00:00Z",
			"scheduled_until": "2026-02-15T04:00:00Z",
			"components": [],
			"incident_updates": []
		}`
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(scheduledInProgressJSON)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"page":            "kctbh9vrtdwd",
				"incidentType":    "scheduled",
				"name":            "Maintenance In Progress",
				"scheduledFor":    "2026-02-15T02:00:00Z",
				"scheduledUntil":  "2026-02-15T04:00:00Z",
				"statusScheduled": "in_progress",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.Equal(t, "statuspage.incident", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "sched456", data["id"])
		assert.Equal(t, "in_progress", data["status"])
	})

	t.Run("success with components sends per-component status", func(t *testing.T) {
		componentsJSON := `[{"id":"comp1","name":"comp1","status":"operational"},{"id":"comp2","name":"comp2","status":"operational"}]`
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(componentsJSON))},
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(sampleIncidentJSON)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"page":         "kctbh9vrtdwd",
				"incidentType": "realtime",
				"name":         "Test with components",
				"components": []any{
					map[string]any{"componentId": "comp1", "status": "degraded_performance"},
					map[string]any{"componentId": "comp2", "status": "major_outage"},
				},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2) // ListComponents, then CreateIncident
		reqBody, readErr := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, readErr)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(reqBody, &payload))
		incident, ok := payload["incident"].(map[string]any)
		require.True(t, ok)
		components, ok := incident["components"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "degraded_performance", components["comp1"])
		assert.Equal(t, "major_outage", components["comp2"])
	})

	t.Run("resolves component names to IDs before API call", func(t *testing.T) {
		// User enters "Management Portal" (name) but API expects ID "mgmt123"
		componentsJSON := `[{"id":"mgmt123","name":"Management Portal","status":"operational"},{"id":"api456","name":"API","status":"operational"}]`
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(componentsJSON))},
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(sampleIncidentJSON))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"page":         "kctbh9vrtdwd",
				"incidentType": "realtime",
				"name":         "Test with component names",
				"components": []any{
					map[string]any{"componentId": "Management Portal", "status": "partial_outage"},
					map[string]any{"componentId": "API", "status": "degraded_performance"},
				},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		reqBody, readErr := io.ReadAll(httpContext.Requests[1].Body)
		require.NoError(t, readErr)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(reqBody, &payload))
		incident, ok := payload["incident"].(map[string]any)
		require.True(t, ok)
		componentIDs, ok := incident["component_ids"].([]any)
		require.True(t, ok)
		assert.Contains(t, componentIDs, "mgmt123")
		assert.Contains(t, componentIDs, "api456")
		components, ok := incident["components"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "partial_outage", components["mgmt123"])
		assert.Equal(t, "degraded_performance", components["api456"])
	})

	t.Run("API error returns error and no emit", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Validation failed"}`)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"page":         "kctbh9vrtdwd",
				"incidentType": "realtime",
				"name":         "Test",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Empty(t, executionState.Payloads)
	})
}
