package firehydrant

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

func Test__CreateIncident__Setup(t *testing.T) {
	component := &CreateIncident{}

	t.Run("valid configuration -> success", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: CreateIncidentSpec{
				Name:        "Test Incident",
				Description: "Test description",
				Severity:    "sev1",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("missing name -> error", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: CreateIncidentSpec{
				Name: "",
			},
			Metadata: metadataCtx,
		})

		require.ErrorContains(t, err, "name is required")
	})
}

func Test__CreateIncident__Execute(t *testing.T) {
	component := &CreateIncident{}

	t.Run("successful creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`
						{
							"data": {
								"id": "inc-123",
								"type": "incident",
								"attributes": {
									"name": "Database Outage",
									"description": "Database is down",
									"severity": "sev1",
									"status": "started",
									"created_at": "2026-01-19T12:00:00Z",
									"started_at": "2026-01-19T12:00:00Z",
									"resolved_at": null,
									"archived_at": null,
									"html_url": "https://app.firehydrant.io/incidents/inc-123"
								}
							}
						}
					`)),
				},
			},
		}

		executionStateCtx := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: CreateIncidentSpec{
				Name:        "Database Outage",
				Description: "Database is down",
				Severity:    "sev1",
			},
			HTTP:           httpContext,
			ExecutionState: executionStateCtx,
			Integration:    integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		// Verify request
		req := httpContext.Requests[0]
		assert.Equal(t, "https://api.firehydrant.io/v1/incidents", req.URL.String())
		assert.Equal(t, http.MethodPost, req.Method)

		// Verify emitted event
		require.Len(t, executionStateCtx.Payloads, 1)
		assert.Equal(t, "firehydrant.incident", executionStateCtx.Type)
	})

	t.Run("API error -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error": "invalid request"}`)),
				},
			},
		}

		executionStateCtx := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: CreateIncidentSpec{
				Name: "Test Incident",
			},
			HTTP:           httpContext,
			ExecutionState: executionStateCtx,
			Integration:    integrationCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create incident")
		assert.Len(t, executionStateCtx.Payloads, 0)
	})
}
