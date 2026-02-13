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

func Test__GetIncident__Setup(t *testing.T) {
	component := &GetIncident{}

	t.Run("valid configuration with incidentId", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "abc123-def456",
			},
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

	t.Run("invalid configuration format -> decode error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: "invalid-config",
		})

		require.ErrorContains(t, err, "error decoding configuration")
	})
}

func Test__GetIncident__Execute(t *testing.T) {
	component := &GetIncident{}

	t.Run("successful get emits incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "inc-uuid-123",
							"type": "incidents",
							"attributes": {
								"title": "Database connection issues",
								"sequential_id": 42,
								"slug": "database-connection-issues",
								"summary": "Users are experiencing slow database queries.",
								"status": "started",
								"severity": "sev1",
								"started_at": "2026-01-19T12:00:00Z",
								"updated_at": "2026-01-19T12:00:00Z",
								"url": "https://app.rootly.com/incidents/inc-uuid-123"
							}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "inc-uuid-123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, execState.Channel)
		assert.Equal(t, "rootly.incident", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		// Verify request
		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Contains(t, req.URL.String(), "/incidents/inc-uuid-123")
		assert.Equal(t, "application/vnd.api+json", req.Header.Get("Content-Type"))
	})

	t.Run("API error returns error and does not emit", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errors": [{"title": "Record not found"}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey": "test-api-key",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "nonexistent-id",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get incident")
		assert.False(t, execState.Passed)
		assert.Empty(t, execState.Channel)
	})
}
