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

func Test__CreateCheckRule__Setup(t *testing.T) {
	component := CreateCheckRule{}

	t.Run("name is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("expression is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"name": "Test Rule"},
		})

		require.ErrorContains(t, err, "expression is required")
	})

	t.Run("threshold required when using $__threshold in expression", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"name":       "Test Rule",
				"expression": "sum(rate(http_requests_total[5m])) > $__threshold",
			},
		})

		require.ErrorContains(t, err, "at least one threshold (degraded or critical) is required when using $__threshold")
	})

	t.Run("valid setup with thresholds", func(t *testing.T) {
		degraded := 0.5
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"name":       "Test Rule",
				"expression": "sum(rate(http_requests_total[5m])) > $__threshold",
				"thresholds": map[string]any{
					"degraded": &degraded,
				},
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid setup without $__threshold", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"name":       "Test Rule",
				"expression": "sum(rate(http_requests_total[5m])) > 0.5",
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateCheckRule__Execute(t *testing.T) {
	component := CreateCheckRule{}

	t.Run("successful creation", func(t *testing.T) {
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
									"name": "Test Alert",
									"labels": {
										"dash0.com/dataset": "default",
										"dash0.com/id": "test-rule-id-123"
									}
								},
								"spec": {
									"name": "Test Alert",
									"expression": "up == 0",
									"enabled": true
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
				"name":       "Test Alert",
				"expression": "up == 0",
				"dataset":    "default",
				"enabled":    true,
			},
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "default", execCtx.Channel)
		assert.Equal(t, "dash0.checkRule.created", execCtx.Type)
		require.Len(t, execCtx.Payloads, 1)
	})

	t.Run("API error handling", func(t *testing.T) {
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusBadRequest,
						Body: io.NopCloser(strings.NewReader(`
							{
								"error": "invalid expression syntax"
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
				"name":       "Test Alert",
				"expression": "invalid{{syntax",
				"dataset":    "default",
				"enabled":    true,
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create check rule")
	})
}
