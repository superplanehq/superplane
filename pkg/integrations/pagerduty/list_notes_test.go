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

func Test__ListNotes__Setup(t *testing.T) {
	component := &ListNotes{}

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

func Test__ListNotes__Execute(t *testing.T) {
	component := &ListNotes{}

	t.Run("successfully lists notes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"notes": [
								{
									"id": "PVL9NF8",
									"user": {
										"id": "PLH1HKV",
										"type": "user_reference",
										"summary": "John Smith",
										"html_url": "https://acme.pagerduty.com/users/PLH1HKV"
									},
									"channel": {
										"type": "web_ui"
									},
									"content": "Investigation started. Checking server logs.",
									"created_at": "2024-01-15T10:30:00Z"
								},
								{
									"id": "PVL9NF9",
									"user": {
										"id": "PLH1HKV",
										"type": "user_reference",
										"summary": "John Smith",
										"html_url": "https://acme.pagerduty.com/users/PLH1HKV"
									},
									"channel": {
										"type": "web_ui"
									},
									"content": "Root cause identified. Deploying fix.",
									"created_at": "2024-01-15T10:45:00Z"
								}
							]
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
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "pagerduty.notes.list", execCtx.Type)
		assert.Equal(t, core.DefaultOutputChannel.Name, execCtx.Channel)

		// Verify the request was made correctly
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/incidents/PT4KHLK/notes")

		// Verify response contains expected data
		require.Len(t, execCtx.Payloads, 1)
		wrappedPayload, ok := execCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		responseData, ok := wrappedPayload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 2, responseData["total"])
	})

	t.Run("successfully lists empty notes", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"notes": []
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
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)

		// Verify response contains expected data
		require.Len(t, execCtx.Payloads, 1)
		wrappedPayload, ok := execCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		responseData, ok := wrappedPayload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 0, responseData["total"])
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
		assert.Contains(t, err.Error(), "failed to list notes")
	})
}
