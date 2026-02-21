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
				"fromEmail":  "user@example.com",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with resolution", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
				"fromEmail":  "user@example.com",
				"resolution": "Deployed fix in v1.2.3",
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
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("empty incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "",
				"fromEmail":  "user@example.com",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("missing fromEmail returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "fromEmail is required")
	})

	t.Run("empty fromEmail returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
				"fromEmail":  "",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "fromEmail is required")
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
								"title": "The server is on fire.",
								"status": "resolved",
								"html_url": "https://subdomain.pagerduty.com/incidents/PT4KHLK",
								"incident_number": 1234,
								"created_at": "2015-10-06T21:30:42Z",
								"updated_at": "2015-10-06T21:40:23Z",
								"resolved_at": "2015-10-06T21:40:23Z"
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
		assert.Equal(t, ResolveChannelSuccess, execCtx.Channel)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/incidents/PT4KHLK")
		assert.Equal(t, "user@example.com", httpContext.Requests[0].Header.Get("From"))
	})

	t.Run("successfully resolves incident with resolution note", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"incident": {
								"id": "PT4KHLK",
								"type": "incident",
								"title": "The server is on fire.",
								"status": "resolved",
								"html_url": "https://subdomain.pagerduty.com/incidents/PT4KHLK",
								"incident_number": 1234,
								"created_at": "2015-10-06T21:30:42Z",
								"updated_at": "2015-10-06T21:40:23Z",
								"resolved_at": "2015-10-06T21:40:23Z"
							}
						}
					`)),
				},
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`
						{
							"note": {
								"id": "PVL9NF8",
								"content": "Deployed fix in v1.2.3"
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
				"resolution": "Deployed fix in v1.2.3",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ResolveChannelSuccess, execCtx.Channel)

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/incidents/PT4KHLK")
		assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/incidents/PT4KHLK/notes")
	})

	t.Run("incident not found emits to failed channel", func(t *testing.T) {
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

		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"incidentId": "INVALID",
				"fromEmail":  "user@example.com",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ResolveChannelFailed, execCtx.Channel)
		assert.Equal(t, "pagerduty.incident", execCtx.Type)

		require.Len(t, execCtx.Payloads, 1)
		wrappedPayload, ok := execCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := wrappedPayload["data"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "INVALID", data["incidentId"])
		assert.NotEmpty(t, data["error"])
	})

	t.Run("invalid user emits to failed channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error": {"message": "Invalid user"}}`)),
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
				"fromEmail":  "invalid@example.com",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, ResolveChannelFailed, execCtx.Channel)
	})
}
