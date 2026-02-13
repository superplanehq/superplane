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

func Test__ReassignEscalationPolicy__Setup(t *testing.T) {
	component := &ReassignEscalationPolicy{}

	t.Run("valid configuration", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":       "PT4KHLK",
				"escalationPolicy": "PNEWPOL",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with fromEmail", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":       "PT4KHLK",
				"escalationPolicy": "PNEWPOL",
				"fromEmail":        "user@example.com",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("missing incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"escalationPolicy": "PNEWPOL",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("empty incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":       "",
				"escalationPolicy": "PNEWPOL",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("missing escalationPolicy returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "escalationPolicy is required")
	})

	t.Run("empty escalationPolicy returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":       "PT4KHLK",
				"escalationPolicy": "",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "escalationPolicy is required")
	})
}

func Test__ReassignEscalationPolicy__Execute(t *testing.T) {
	component := &ReassignEscalationPolicy{}

	t.Run("successfully reassigns escalation policy", func(t *testing.T) {
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
								"status": "triggered",
								"urgency": "high",
								"escalation_policy": {
									"id": "PNEWPOL",
									"type": "escalation_policy_reference",
									"summary": "Engineering Managers On-Call"
								}
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
				"incidentId":       "PT4KHLK",
				"escalationPolicy": "PNEWPOL",
				"fromEmail":        "user@example.com",
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

		// Verify request body contains escalation policy
		body, _ := io.ReadAll(httpContext.Requests[0].Body)
		assert.Contains(t, string(body), "PNEWPOL")
		assert.Contains(t, string(body), "escalation_policy")
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
				"incidentId":       "INVALID",
				"escalationPolicy": "PNEWPOL",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to reassign escalation policy")
	})
}
