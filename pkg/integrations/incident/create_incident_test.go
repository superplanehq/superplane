package incident

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateIncident__Setup(t *testing.T) {
	component := &CreateIncident{}

	t.Run("valid configuration", func(t *testing.T) {
		severitiesResp := `{"severities":[{"id":"sev_abc123","name":"Minor","description":"","rank":1}]}`
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":       "Test Incident",
				"summary":    "Test summary",
				"severityId": "sev_abc123",
				"visibility": "public",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(severitiesResp)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"summary": "Test summary",
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("empty name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":       "",
				"summary":    "Test summary",
				"severityId": "sev_abc123",
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing severity returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name": "Minimal Incident",
			},
		})

		require.ErrorContains(t, err, "severity is required")
	})

	t.Run("name and severity - minimal required fields", func(t *testing.T) {
		severitiesResp := `{"severities":[{"id":"sev_abc123","name":"Minor","description":"","rank":1}]}`
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":       "Minimal Incident",
				"severityId": "sev_abc123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(severitiesResp)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
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

	t.Run("severity not in list returns error", func(t *testing.T) {
		severitiesResp := `{"severities":[{"id":"sev_other","name":"Major","description":"","rank":2}]}`
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":       "Test Incident",
				"severityId": "sev_abc123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(severitiesResp)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiKey": "test-key"},
			},
		})

		require.Error(t, err)
		require.ErrorContains(t, err, "not found or no longer available")
	})
}

func Test__CreateIncident__Execute(t *testing.T) {
	component := &CreateIncident{}

	createIncidentResponse := `{"incident":{"id":"01FDAG4SAP5TYPT98WGR2N7W91","name":"Database issues","summary":"Slow queries","reference":"INC-123","permalink":"https://app.incident.io/incidents/123","visibility":"public","created_at":"2021-08-17T13:28:57Z","updated_at":"2021-08-17T13:28:57Z"}}`

	t.Run("success creates incident and emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(createIncidentResponse)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execStateCtx := &contexts.ExecutionStateContext{}
		execID := uuid.New()

		err := component.Execute(core.ExecutionContext{
			ID:             execID,
			Configuration:  map[string]any{"name": "Database issues", "summary": "Slow queries", "severityId": "sev_abc123", "visibility": "public"},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execStateCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://api.incident.io/v2/incidents", req.URL.String())
		assert.True(t, execStateCtx.Passed)
		require.Len(t, execStateCtx.Payloads, 1)
		payload := execStateCtx.Payloads[0].(map[string]any)
		assert.Equal(t, "incident.incident", payload["type"])
		assert.NotNil(t, payload["data"])
		// payload["data"] is *IncidentV2 from the client
		data, ok := payload["data"].(*IncidentV2)
		require.True(t, ok)
		assert.Equal(t, "01FDAG4SAP5TYPT98WGR2N7W91", data.ID)
		assert.Equal(t, "Database issues", data.Name)
		assert.Equal(t, "INC-123", data.Reference)
	})
}
