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

func Test__EscalateIncident__Setup(t *testing.T) {
	component := &EscalateIncident{}

	t.Run("valid configuration", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":      "PT4KHLK",
				"escalationLevel": "2",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with all fields", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":      "PT4KHLK",
				"fromEmail":       "user@example.com",
				"escalationLevel": "5",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("missing incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"escalationLevel": "2",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("empty incidentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":      "",
				"escalationLevel": "2",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "incidentId is required")
	})

	t.Run("missing escalationLevel returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId": "PT4KHLK",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "escalationLevel is required")
	})

	t.Run("empty escalationLevel returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":      "PT4KHLK",
				"escalationLevel": "",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "escalationLevel is required")
	})

	t.Run("escalation level above 10 returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":      "PT4KHLK",
				"escalationLevel": "11",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "escalationLevel must be between 1 and 10")
	})

	t.Run("escalation level of 0 returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":      "PT4KHLK",
				"escalationLevel": "0",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "escalationLevel must be between 1 and 10")
	})

	t.Run("escalation level of 1 is valid (min level)", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":      "PT4KHLK",
				"escalationLevel": "1",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("escalation level of 10 is valid (max level)", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":      "PT4KHLK",
				"escalationLevel": "10",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
	})

	t.Run("invalid escalation level string returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"incidentId":      "PT4KHLK",
				"escalationLevel": "invalid",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "invalid escalationLevel")
	})
}

func Test__EscalateIncident__Execute(t *testing.T) {
	component := &EscalateIncident{}

	t.Run("successfully escalates incident to specific level", func(t *testing.T) {
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
								"urgency": "high"
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
				"incidentId":      "PT4KHLK",
				"escalationLevel": "3",
				"fromEmail":       "user@example.com",
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
		assert.Equal(t, "user@example.com", httpContext.Requests[0].Header.Get("From"))

		// Verify request body contains escalation level
		body, _ := io.ReadAll(httpContext.Requests[0].Body)
		assert.Contains(t, string(body), "escalation_level")
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"error": {"message": "Cannot Escalate", "code": 2001}}`)),
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
				"incidentId":      "INVALID",
				"escalationLevel": "2",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execCtx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to escalate incident")
	})
}
