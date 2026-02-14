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
				"incidentId": "PT4KHLK",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with all optional fields", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
				"fromEmail":  "user@example.com",
				"resolution": "Fixed the issue by restarting the server",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("missing incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("empty incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})
}

func Test__ResolveIncident__Execute(t *testing.T) {
	component := &ResolveIncident{}

	t.Run("successfully resolves incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"incident": {
								"id": "PT4KHLK",
								"type": "incident",
								"title": "Server is on fire",
								"status": "resolved",
								"urgency": "high",
								"resolved_at": "2024-01-15T12:00:00Z"
							}
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
				"fromEmail":  "user@example.com",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "pagerduty.incident", execCtx.Type)
		assert.Equal(t, core.DefaultOutputChannel.Name, execCtx.Channel)

		// Verify the request was made correctly
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/incidents/PT4KHLK")
	})

	t.Run("successfully resolves incident with resolution notes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"incident": {
								"id": "PT4KHLK",
								"type": "incident",
								"title": "Server is on fire",
								"status": "resolved",
								"urgency": "high",
								"resolved_at": "2024-01-15T12:00:00Z"
							}
						}
					`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
				"resolution": "Fixed by restarting the server",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)

		// Verify request body contains resolution
		require.Len(t, httpContext.Requests, 1)
		body, _ := io.ReadAll(httpContext.Requests[0].Body)
		assert.Contains(t, string(body), "Fixed by restarting the server")
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "Incident not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAPIToken,
				"apiToken": "test-token",
			},
		}

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "INVALID",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve incident")
	})
}
