package dash0

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

func Test__UpdateCheckRule__Setup(t *testing.T) {
	component := UpdateCheckRule{}

	t.Run("checkRuleId is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "checkRuleId is required")
	})

	t.Run("name is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"checkRuleId": "test-rule-123"},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("expression is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"checkRuleId": "test-rule-123",
				"name":        "Test Rule",
			},
		})

		require.ErrorContains(t, err, "expression is required")
	})

	t.Run("threshold required when using $__threshold", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"checkRuleId": "test-rule-123",
				"name":        "Test Rule",
				"expression":  "sum(rate(http_requests_total[5m])) > $__threshold",
			},
		})

		require.ErrorContains(t, err, "at least one threshold (degraded or critical) is required when using $__threshold")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"checkRuleId": "test-rule-123",
				"name":        "Test Rule",
				"expression":  "sum(rate(http_requests_total[5m])) > 0.5",
			},
		})

		require.NoError(t, err)
	})
}

func Test__UpdateCheckRule__Execute(t *testing.T) {
	component := UpdateCheckRule{}

	t.Run("successful update", func(t *testing.T) {
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`
							{
								"kind": "Dash0CheckRule",
								"metadata": {
									"name": "Updated Alert",
									"labels": {
										"dash0.com/dataset": "default",
										"dash0.com/id": "test-rule-id-123",
										"dash0.com/version": "2"
									}
								},
								"spec": {
									"name": "Updated Alert",
									"expression": "up == 0",
									"enabled": false
								}
							}
						`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
			Configuration: map[string]any{
				"checkRuleId": "test-rule-id-123",
				"name":        "Updated Alert",
				"expression":  "up == 0",
				"dataset":     "default",
				"enabled":     false,
			},
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "default", execCtx.Channel)
		assert.Equal(t, "dash0.checkRule.updated", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("check rule not found", func(t *testing.T) {
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusNotFound,
						Body: io.NopCloser(strings.NewReader(`
							{
								"error": "check rule not found"
							}
						`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://api.us-west-2.aws.dash0.com",
				},
			},
			ExecutionState: execCtx,
			Configuration: map[string]any{
				"checkRuleId": "non-existent-rule",
				"name":        "Test Rule",
				"expression":  "up == 0",
				"dataset":     "default",
				"enabled":     true,
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update check rule")
	})
}
