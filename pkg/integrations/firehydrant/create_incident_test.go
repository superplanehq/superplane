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
						"severity": {"slug": "SEV1", "description": "Critical"},
						"priority": {"slug": "P1", "description": "Highest"}
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
}
