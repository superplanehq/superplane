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

func Test__ResolveIncident__Setup(t *testing.T) {
	component := &ResolveIncident{}

	t.Run("valid configuration", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "P1ABC23",
				"fromEmail":  "user@example.com",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
		metadata := metadataCtx.Metadata.(NodeMetadata)
		assert.Nil(t, metadata.Service)
	})

	t.Run("valid configuration with resolution note", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "P1ABC23",
				"fromEmail":  "user@example.com",
				"resolution": "Fixed by deploying v2.1.0",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("missing incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"fromEmail": "user@example.com",
			},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("empty incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "",
				"fromEmail":  "user@example.com",
			},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})
}

func Test__ResolveIncident__Execute(t *testing.T) {
	component := &ResolveIncident{}

	t.Run("resolves incident without resolution note", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"incident": {
							"id": "P1ABC23",
							"status": "resolved",
							"title": "Server issue",
							"html_url": "https://example.pagerduty.com/incidents/P1ABC23"
						}
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

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "P1ABC23",
				"fromEmail":  "user@example.com",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "pagerduty.incident", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Equal(t, "https://api.pagerduty.com/incidents/P1ABC23", httpContext.Requests[0].URL.String())
		assert.Equal(t, "user@example.com", httpContext.Requests[0].Header.Get("From"))
	})

	t.Run("resolves incident with resolution note", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"incident": {
							"id": "P1ABC23",
							"status": "resolved",
							"title": "Server issue",
							"html_url": "https://example.pagerduty.com/incidents/P1ABC23"
						}
					}`)),
				},
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"note": {
							"id": "PNOTE01",
							"content": "Fixed by deploying v2.1.0"
						}
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

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "P1ABC23",
				"fromEmail":  "user@example.com",
				"resolution": "Fixed by deploying v2.1.0",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "pagerduty.incident", executionState.Type)

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Equal(t, "https://api.pagerduty.com/incidents/P1ABC23", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
		assert.Equal(t, "https://api.pagerduty.com/incidents/P1ABC23/notes", httpContext.Requests[1].URL.String())
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": {"message": "Incident not found"}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "INVALID",
				"fromEmail":  "user@example.com",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve incident")
	})

	t.Run("note creation failure returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"incident": {
							"id": "P1ABC23",
							"status": "resolved",
							"title": "Server issue"
						}
					}`)),
				},
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error": {"message": "Invalid note"}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "P1ABC23",
				"fromEmail":  "user@example.com",
				"resolution": "Fixed the issue",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve incident")
		assert.Contains(t, err.Error(), "resolution note")
	})
}
