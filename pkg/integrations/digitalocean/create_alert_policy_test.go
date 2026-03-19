package digitalocean

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

func Test__CreateAlertPolicy__Setup(t *testing.T) {
	component := &CreateAlertPolicy{}

	t.Run("missing description returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"type":    "v1/insights/droplet/cpu",
				"compare": "GreaterThan",
				"value":   75,
				"window":  "5m",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "description is required")
	})

	t.Run("missing type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"description": "High CPU alert",
				"compare":     "GreaterThan",
				"value":       75,
				"window":      "5m",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "type is required")
	})

	t.Run("missing compare returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"description": "High CPU alert",
				"type":        "v1/insights/droplet/cpu",
				"value":       75,
				"window":      "5m",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "compare is required")
	})

	t.Run("missing window returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"description": "High CPU alert",
				"type":        "v1/insights/droplet/cpu",
				"compare":     "GreaterThan",
				"value":       75,
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "window is required")
	})

	t.Run("slack channel without slack url returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"description":  "High CPU alert",
				"type":         "v1/insights/droplet/cpu",
				"compare":      "GreaterThan",
				"value":        75,
				"window":       "5m",
				"slackChannel": "#alerts",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "slackChannel and slackUrl must both be provided together")
	})

	t.Run("slack url without slack channel returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"description": "High CPU alert",
				"type":        "v1/insights/droplet/cpu",
				"compare":     "GreaterThan",
				"value":       75,
				"window":      "5m",
				"slackUrl":    "https://hooks.slack.com/services/test",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "slackChannel and slackUrl must both be provided together")
	})

	t.Run("no notification channel returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"description": "High CPU alert",
				"type":        "v1/insights/droplet/cpu",
				"compare":     "GreaterThan",
				"value":       75,
				"window":      "5m",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "at least one notification channel (email or Slack) is required")
	})

	t.Run("valid configuration with email -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"description": "High CPU alert",
				"type":        "v1/insights/droplet/cpu",
				"compare":     "GreaterThan",
				"value":       75,
				"window":      "5m",
				"email":       []any{"ops@example.com"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with both slack fields -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"description":  "High CPU alert",
				"type":         "v1/insights/droplet/cpu",
				"compare":      "GreaterThan",
				"value":        75,
				"window":       "5m",
				"slackChannel": "#alerts",
				"slackUrl":     "https://hooks.slack.com/services/test",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__CreateAlertPolicy__Execute(t *testing.T) {
	component := &CreateAlertPolicy{}

	t.Run("successful creation -> emits policy data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"policy": {
							"uuid": "669adfc8-d72b-4d2d-80ed-bea78d6e1562",
							"description": "High CPU alert",
							"type": "v1/insights/droplet/cpu",
							"compare": "GreaterThan",
							"value": 75,
							"window": "5m",
							"entities": [],
							"tags": [],
							"alerts": {"slack": [], "email": ["ops@example.com"]},
							"enabled": true
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"description": "High CPU alert",
				"type":        "v1/insights/droplet/cpu",
				"compare":     "GreaterThan",
				"value":       75,
				"window":      "5m",
				"email":       []any{"ops@example.com"},
				"enabled":     true,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.alertpolicy.created", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("creation with slack notification -> emits policy data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"policy": {
							"uuid": "669adfc8-d72b-4d2d-80ed-bea78d6e1562",
							"description": "High CPU alert",
							"type": "v1/insights/droplet/cpu",
							"compare": "GreaterThan",
							"value": 75,
							"window": "5m",
							"entities": [],
							"tags": [],
							"alerts": {
								"slack": [{"url": "https://hooks.slack.com/services/test", "channel": "#alerts"}],
								"email": []
							},
							"enabled": true
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"description":  "High CPU alert",
				"type":         "v1/insights/droplet/cpu",
				"compare":      "GreaterThan",
				"value":        75,
				"window":       "5m",
				"slackChannel": "#alerts",
				"slackUrl":     "https://hooks.slack.com/services/test",
				"enabled":      true,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.alertpolicy.created", executionState.Type)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unprocessable_entity","message":"The type field is invalid."}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"description": "Bad alert",
				"type":        "invalid/type",
				"compare":     "GreaterThan",
				"value":       75,
				"window":      "5m",
				"email":       []any{"ops@example.com"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create alert policy")
		assert.False(t, executionState.Passed)
	})
}
