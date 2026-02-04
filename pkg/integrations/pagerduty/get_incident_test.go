package pagerduty

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

func Test__GetIncident__Setup(t *testing.T) {
	component := &GetIncident{}

	t.Run("valid configuration", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("missing incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("empty incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "",
			},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})
}

func Test__GetIncident__Execute(t *testing.T) {
	component := &GetIncident{}

	t.Run("successfully gets incident with all related data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Incident response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"incident": {
							"id": "PT4KHLK",
							"incident_number": 1234,
							"title": "The server is on fire.",
							"status": "triggered",
							"urgency": "high"
						}
					}`)),
				},
				// Alerts response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"alerts": [
							{
								"id": "PT4KHLK:0",
								"summary": "CPU usage exceeded 90%"
							}
						]
					}`)),
				},
				// Notes response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"notes": [
							{
								"id": "PNQ2YA7",
								"content": "Investigating the issue"
							}
						]
					}`)),
				},
				// Log entries response
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"log_entries": [
							{
								"id": "Q02JTSNZWHSEKV",
								"type": "trigger_log_entry"
							}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionState{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 4)
		require.Len(t, executionState.Emitted, 1)

		emitted := executionState.Emitted[0]
		assert.Equal(t, "default", emitted.OutputChannel)
		assert.Equal(t, "pagerduty.incident", emitted.Type)

		data := emitted.Data[0].(map[string]any)
		assert.NotNil(t, data["incident"])
		assert.NotNil(t, data["alerts"])
		assert.NotNil(t, data["notes"])
		assert.NotNil(t, data["log_entries"])
	})

	t.Run("returns incident even when optional calls fail", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Incident response - success
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"incident": {
							"id": "PT4KHLK",
							"incident_number": 1234,
							"title": "The server is on fire.",
							"status": "triggered",
							"urgency": "high"
						}
					}`)),
				},
				// Alerts response - fails
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Access denied"}`)),
				},
				// Notes response - fails
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Access denied"}`)),
				},
				// Log entries response - fails
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Access denied"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionState{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, executionState.Emitted, 1)

		emitted := executionState.Emitted[0]
		data := emitted.Data[0].(map[string]any)
		assert.NotNil(t, data["incident"])
		// Optional data should be nil when API calls fail
		assert.Nil(t, data["alerts"])
		assert.Nil(t, data["notes"])
		assert.Nil(t, data["log_entries"])
	})

	t.Run("returns error when incident call fails", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Incident response - fails
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Incident not found"}`)),
				},
				// Alerts response
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"alerts": []}`)),
				},
				// Notes response
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"notes": []}`)),
				},
				// Log entries response
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"log_entries": []}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionState{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "INVALID",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get incident")
	})
}
