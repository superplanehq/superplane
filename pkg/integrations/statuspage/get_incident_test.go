package statuspage

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

const sampleGetIncidentJSON = `{
	"id": "p31zjtct2jer",
	"name": "Database Connection Issues",
	"status": "investigating",
	"impact": "major",
	"shortlink": "http://stspg.io/p31zjtct2jer",
	"created_at": "2026-02-12T10:30:00.000Z",
	"updated_at": "2026-02-12T10:45:00.000Z",
	"page_id": "kctbh9vrtdwd",
	"components": [],
	"incident_updates": [
		{"id": "upd1", "status": "investigating", "body": "We are investigating.", "created_at": "2026-02-12T10:30:00.000Z"},
		{"id": "upd2", "status": "identified", "body": "Root cause identified.", "created_at": "2026-02-12T10:45:00.000Z"}
	]
}`

func Test__GetIncident__Setup(t *testing.T) {
	component := &GetIncident{}

	t.Run("valid configuration", func(t *testing.T) {
		// Setup verifies incident exists when static; mock GetIncident response.
		incidentResponse := `{"incident":` + sampleGetIncidentJSON + `}`
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
				"page":     "kctbh9vrtdwd",
				"incident": "p31zjtct2jer",
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

	t.Run("missing page returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incident": "p31zjtct2jer",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page is required")
	})

	t.Run("missing incident returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page": "kctbh9vrtdwd",
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "incident is required")
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
				"page":     "kctbh9vrtdwd",
				"incident": "nonexistent",
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
				"page":     "kctbh9vrtdwd",
				"incident": "{{ $['Create Incident'].data.id }}",
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})

	t.Run("use expression without incidentExpression returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"page":     "kctbh9vrtdwd",
				"incident": IncidentUseExpressionID,
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
			},
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__GetIncident__Execute(t *testing.T) {
	component := &GetIncident{}

	t.Run("success emits incident with incident_updates in API order", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(sampleGetIncidentJSON)),
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
		assert.Equal(t, "http://stspg.io/p31zjtct2jer", data["shortlink"])

		updates, ok := data["incident_updates"].([]any)
		require.True(t, ok)
		require.Len(t, updates, 2)
		upd1, ok := updates[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "upd1", upd1["id"])
		assert.Equal(t, "We are investigating.", upd1["body"])
		upd2, ok := updates[1].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "upd2", upd2["id"])
		assert.Equal(t, "Root cause identified.", upd2["body"])
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
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get incident")
		assert.Empty(t, executionState.Payloads)
	})

	t.Run("use expression with incidentExpression uses resolved value", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(sampleGetIncidentJSON))},
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
				"incidentExpression": "p31zjtct2jer", // resolved at runtime from expression
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
}
