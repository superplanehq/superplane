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

func Test__UpdateIncident__Setup(t *testing.T) {
	component := &UpdateIncident{}

	t.Run("valid configuration with status and body", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":     "kctbh9vrtdwd",
				"incident": "p31zjtct2jer",
				"status":   "resolved",
				"body":     "Issue resolved.",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("valid configuration with status only", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":     "kctbh9vrtdwd",
				"incident": "p31zjtct2jer",
				"status":   "in_progress",
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
				"page":            "kctbh9vrtdwd",
				"incident":        "p31zjtct2jer",
				"status":          "resolved",
				"componentIds":    []string{"comp1"},
				"componentStatus": "operational",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("missing page returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incident": "p31zjtct2jer",
				"status":   "resolved",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page is required")
	})

	t.Run("missing incident returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":   "kctbh9vrtdwd",
				"status": "resolved",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incident is required")
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
}

func Test__UpdateIncident__Execute(t *testing.T) {
	component := &UpdateIncident{}

	t.Run("success emits incident", func(t *testing.T) {
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
				"status":   "resolved",
				"body":     "All systems operational.",
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

	t.Run("deliver_notifications false is sent in request", func(t *testing.T) {
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
		deliverFalse := false

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"page":                 "kctbh9vrtdwd",
				"incident":             "p31zjtct2jer",
				"status":               "resolved",
				"deliverNotifications": &deliverFalse,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		reqBody, readErr := io.ReadAll(httpContext.Requests[0].Body)
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
				"page":     "kctbh9vrtdwd",
				"incident": "nonexistent",
				"status":   "resolved",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update incident")
		assert.Empty(t, executionState.Payloads)
	})
}
