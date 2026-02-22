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

const sampleUpdatedIncidentJSON = `{
	"id": "p31zjtct2jer",
	"name": "Database Connection Issues",
	"status": "resolved",
	"impact": "major",
	"shortlink": "http://stspg.io/p31zjtct2jer",
	"created_at": "2026-02-12T10:30:00.000Z",
	"updated_at": "2026-02-12T11:00:00.000Z",
	"page_id": "kctbh9vrtdwd",
	"components": [],
	"incident_updates": []
}`

const sampleRealtimeIncidentJSON = `{
	"id": "p31zjtct2jer",
	"name": "Database Connection Issues",
	"status": "investigating",
	"impact": "major",
	"shortlink": "http://stspg.io/p31zjtct2jer",
	"page_id": "kctbh9vrtdwd",
	"components": [],
	"incident_updates": []
}`

const sampleScheduledIncidentJSON = `{
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

func Test__UpdateIncident__Setup(t *testing.T) {
	component := &UpdateIncident{}

	t.Run("valid configuration with status and body", func(t *testing.T) {
		incidentResponse := `{"incident":` + sampleRealtimeIncidentJSON + `}`
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"kctbh9vrtdwd","name":"My Page"}]`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(incidentResponse))},
			},
		}
		metadataCtx := &contexts.MetadataContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incident":       "p31zjtct2jer",
				"incidentType":   "realtime",
				"statusRealtime": "resolved",
				"body":           "Issue resolved.",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})
		require.NoError(t, err)
		md, ok := metadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "Database Connection Issues", md.IncidentName)
	})

	t.Run("valid configuration with status only", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":            "kctbh9vrtdwd",
				"incident":        "p31zjtct2jer",
				"incidentType":    "scheduled",
				"statusScheduled": "in_progress",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("valid configuration with body only", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":     "kctbh9vrtdwd",
				"incident": "p31zjtct2jer",
				"body":     "Update message",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("valid configuration with components", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incident":       "p31zjtct2jer",
				"incidentType":   "realtime",
				"statusRealtime": "resolved",
				"components": []any{
					map[string]any{"componentId": "comp1", "status": "operational"},
				},
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("missing page returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incident":       "p31zjtct2jer",
				"statusRealtime": "resolved",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page is required")
	})

	t.Run("missing incident returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"statusRealtime": "resolved",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incident is required")
	})

	t.Run("invalid incidentType returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incident":       "p31zjtct2jer",
				"incidentType":   "invalid",
				"statusRealtime": "resolved",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incidentType must be realtime or scheduled")
	})

	t.Run("missing status body and components returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":     "kctbh9vrtdwd",
				"incident": "p31zjtct2jer",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one of status, body, impact override, or components must be provided")
	})

	t.Run("incident not found returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"id":"kctbh9vrtdwd","name":"My Page"}]`))},
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":"Not found"}`))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incident":       "nonexistent",
				"incidentType":   "realtime",
				"statusRealtime": "resolved",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incident not found or inaccessible")
	})

	t.Run("expression incident skips verification", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incident":       "{{ $['Create Incident'].data.id }}",
				"incidentType":   "realtime",
				"statusRealtime": "resolved",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("use expression without incidentExpression returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":           "kctbh9vrtdwd",
				"incident":       IncidentUseExpressionID,
				"incidentType":   "realtime",
				"statusRealtime": "resolved",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incident expression is required")
	})

	t.Run("use expression with incidentExpression succeeds", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":               "kctbh9vrtdwd",
				"incident":           IncidentUseExpressionID,
				"incidentExpression": "{{ $['Create Incident'].data.id }}",
				"incidentType":       "realtime",
				"statusRealtime":     "resolved",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__UpdateIncident__Execute(t *testing.T) {
	component := &UpdateIncident{}

	t.Run("success emits incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(sampleRealtimeIncidentJSON)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(sampleUpdatedIncidentJSON)),
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
				"incident":       "p31zjtct2jer",
				"incidentType":   "realtime",
				"statusRealtime": "resolved",
				"body":           "All systems operational.",
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
		assert.Equal(t, "resolved", data["status"])
		assert.Equal(t, "http://stspg.io/p31zjtct2jer", data["shortlink"])
	})

	t.Run("use expression with incidentExpression uses resolved value", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(sampleRealtimeIncidentJSON))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(sampleUpdatedIncidentJSON))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"page":               "kctbh9vrtdwd",
				"incident":           IncidentUseExpressionID,
				"incidentExpression": "p31zjtct2jer",
				"incidentType":       "realtime",
				"statusRealtime":     "resolved",
				"body":               "Updated.",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		wrapped, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrapped["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "p31zjtct2jer", data["id"])
	})

	t.Run("deliver_notifications false is sent in request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(sampleRealtimeIncidentJSON)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(sampleUpdatedIncidentJSON)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		executionState := &contexts.ExecutionStateContext{}
		deliverFalse := false

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"page":                 "kctbh9vrtdwd",
				"incident":             "p31zjtct2jer",
				"incidentType":         "realtime",
				"statusRealtime":       "resolved",
				"deliverNotifications": &deliverFalse,
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
		assert.Equal(t, false, incident["deliver_notifications"])
	})

	t.Run("API error returns error and no emit", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(sampleRealtimeIncidentJSON)),
				},
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Incident not found"}`)),
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
				"incident":       "nonexistent",
				"incidentType":   "realtime",
				"statusRealtime": "resolved",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update incident")
		assert.Empty(t, executionState.Payloads)
	})

	t.Run("type mismatch realtime incident with scheduled status returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(sampleRealtimeIncidentJSON)),
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
				"incident":        "p31zjtct2jer",
				"incidentType":    "scheduled",
				"statusScheduled": "in_progress",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot change a realtime incident to scheduled maintenance")
		assert.Empty(t, executionState.Payloads)
	})

	t.Run("type mismatch scheduled incident with realtime status returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(sampleScheduledIncidentJSON)),
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
				"incident":       "sched123",
				"incidentType":   "realtime",
				"statusRealtime": "resolved",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot change a scheduled maintenance incident to realtime")
		assert.Empty(t, executionState.Payloads)
	})

	t.Run("type match scheduled incident with scheduled status succeeds", func(t *testing.T) {
		scheduledCompletedJSON := `{
			"id": "sched123",
			"name": "Maintenance",
			"status": "completed",
			"impact": "none",
			"shortlink": "http://stspg.io/sched123",
			"page_id": "kctbh9vrtdwd",
			"components": [],
			"incident_updates": []
		}`
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(sampleScheduledIncidentJSON)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(scheduledCompletedJSON)),
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
				"incident":        "sched123",
				"incidentType":    "scheduled",
				"statusScheduled": "completed",
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
		assert.Equal(t, "completed", data["status"])
	})

	t.Run("body only update skips type validation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(sampleUpdatedIncidentJSON)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"page":     "kctbh9vrtdwd",
				"incident": "p31zjtct2jer",
				"body":     "Additional update message.",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "statuspage.incident", executionState.Type)
	})
}
