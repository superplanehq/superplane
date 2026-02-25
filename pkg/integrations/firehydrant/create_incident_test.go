package firehydrant

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateIncident__Configuration(t *testing.T) {
	component := &CreateIncident{}

	t.Run("returns five fields", func(t *testing.T) {
		fields := component.Configuration()
		require.Len(t, fields, 5)
	})

	t.Run("name field is a required string", func(t *testing.T) {
		field := component.Configuration()[0]
		assert.Equal(t, "name", field.Name)
		assert.Equal(t, "Incident Name", field.Label)
		assert.Equal(t, configuration.FieldTypeString, field.Type)
		assert.True(t, field.Required)
		assert.Equal(t, "A succinct name for the incident", field.Description)
	})

	t.Run("summary field is an optional text", func(t *testing.T) {
		field := component.Configuration()[1]
		assert.Equal(t, "summary", field.Name)
		assert.Equal(t, "Summary", field.Label)
		assert.Equal(t, configuration.FieldTypeText, field.Type)
		assert.False(t, field.Required)
	})

	t.Run("description field is an optional text", func(t *testing.T) {
		field := component.Configuration()[2]
		assert.Equal(t, "description", field.Name)
		assert.Equal(t, "Description", field.Label)
		assert.Equal(t, configuration.FieldTypeText, field.Type)
		assert.False(t, field.Required)
	})

	t.Run("severity field is an optional integration resource", func(t *testing.T) {
		field := component.Configuration()[3]
		assert.Equal(t, "severity", field.Name)
		assert.Equal(t, "Severity", field.Label)
		assert.Equal(t, configuration.FieldTypeIntegrationResource, field.Type)
		assert.False(t, field.Required)
		assert.Equal(t, "Select a severity", field.Placeholder)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Resource)
		assert.Equal(t, "severity", field.TypeOptions.Resource.Type)
		assert.True(t, field.TypeOptions.Resource.UseNameAsValue)
	})

	t.Run("priority field is an optional integration resource", func(t *testing.T) {
		field := component.Configuration()[4]
		assert.Equal(t, "priority", field.Name)
		assert.Equal(t, "Priority", field.Label)
		assert.Equal(t, configuration.FieldTypeIntegrationResource, field.Type)
		assert.False(t, field.Required)
		assert.Equal(t, "Select a priority", field.Placeholder)
		require.NotNil(t, field.TypeOptions)
		require.NotNil(t, field.TypeOptions.Resource)
		assert.Equal(t, "priority", field.TypeOptions.Resource.Type)
		assert.True(t, field.TypeOptions.Resource.UseNameAsValue)
	})
}

func Test__CreateIncident__Setup(t *testing.T) {
	component := &CreateIncident{}

	t.Run("valid configuration", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":     "Test Incident",
				"summary":  "Test summary",
				"severity": "SEV1",
			},
		})

		require.NoError(t, err)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"summary":  "Test summary",
				"severity": "SEV1",
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("empty name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":    "",
				"summary": "Test summary",
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("name only - optional fields not required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name": "Minimal Incident",
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

func Test__CreateIncident__Execute(t *testing.T) {
	component := &CreateIncident{}

	t.Run("successful incident creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "04d9fd1a-ba9c-417d-b396-58a6e2c374de",
						"name": "API Outage",
						"number": 42,
						"description": "API is down",
						"summary": "Complete API outage",
						"current_milestone": "started",
						"created_at": "2026-01-19T12:00:00Z",
						"updated_at": "2026-01-19T12:00:00Z",
						"started_at": "2026-01-19T12:00:00Z",
						"severity": "SEV1",
						"priority": "P1"
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":        "API Outage",
				"summary":     "Complete API outage",
				"description": "API is down",
				"severity":    "SEV1",
				"priority":    "P1",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.True(t, executionState.Finished)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "firehydrant.incident", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		// Verify the API request
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "https://api.firehydrant.io/v1/incidents", req.URL.String())
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		assert.Contains(t, req.Header.Get("Authorization"), "Bearer test-api-key")
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error": "unauthorized"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "invalid-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "Test Incident",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create incident")
		assert.False(t, executionState.Passed)
	})

	t.Run("invalid configuration -> decode error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration:  "invalid",
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "error decoding configuration")
	})

	t.Run("missing API key -> client creation error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name": "Test Incident",
			},
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "error creating client")
	})

	t.Run("verifies correct payload structure", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "test-id-123",
						"name": "Database Down",
						"number": 99,
						"summary": "DB unreachable",
						"description": "Primary database is not responding",
						"current_milestone": "started",
						"created_at": "2026-02-25T10:00:00Z",
						"started_at": "2026-02-25T10:00:00Z",
						"severity": "SEV2",
						"priority": "P2"
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-key",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":        "Database Down",
				"summary":     "DB unreachable",
				"description": "Primary database is not responding",
				"severity":    "SEV2",
				"priority":    "P2",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)

		wrappedPayload, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)

		incident, ok := wrappedPayload["data"].(*Incident)
		require.True(t, ok)
		assert.Equal(t, "test-id-123", incident.ID)
		assert.Equal(t, "Database Down", incident.Name)
		assert.Equal(t, 99, incident.Number)
		assert.Equal(t, "DB unreachable", incident.Summary)
		assert.Equal(t, "Primary database is not responding", incident.Description)
		assert.Equal(t, "started", incident.CurrentMilestone)
		assert.Equal(t, "SEV2", incident.Severity)
		assert.Equal(t, "P2", incident.Priority)
	})
}

func Test__CreateIncident__Cancel(t *testing.T) {
	component := &CreateIncident{}

	t.Run("cancel returns no error", func(t *testing.T) {
		err := component.Cancel(core.ExecutionContext{})
		require.NoError(t, err)
	})
}

func Test__CreateIncident__Cleanup(t *testing.T) {
	component := &CreateIncident{}

	t.Run("cleanup returns no error", func(t *testing.T) {
		err := component.Cleanup(core.SetupContext{})
		require.NoError(t, err)
	})
}

func Test__CreateIncident__HandleWebhook(t *testing.T) {
	component := &CreateIncident{}

	t.Run("returns 200 OK", func(t *testing.T) {
		code, err := component.HandleWebhook(core.WebhookRequestContext{})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
	})
}

func Test__CreateIncident__Metadata(t *testing.T) {
	component := &CreateIncident{}

	t.Run("returns correct name", func(t *testing.T) {
		assert.Equal(t, "firehydrant.createIncident", component.Name())
	})

	t.Run("returns correct label", func(t *testing.T) {
		assert.Equal(t, "Create Incident", component.Label())
	})

	t.Run("returns correct description", func(t *testing.T) {
		assert.Equal(t, "Create a new incident in FireHydrant", component.Description())
	})

	t.Run("returns correct icon", func(t *testing.T) {
		assert.Equal(t, "flame", component.Icon())
	})

	t.Run("returns correct color", func(t *testing.T) {
		assert.Equal(t, "gray", component.Color())
	})

	t.Run("returns documentation", func(t *testing.T) {
		docs := component.Documentation()
		assert.NotEmpty(t, docs)
		assert.Contains(t, docs, "Create Incident")
		assert.Contains(t, docs, "Use Cases")
		assert.Contains(t, docs, "Configuration")
		assert.Contains(t, docs, "Output")
	})

	t.Run("returns default output channel", func(t *testing.T) {
		channels := component.OutputChannels(nil)
		require.Len(t, channels, 1)
		assert.Equal(t, core.DefaultOutputChannel, channels[0])
	})
}
