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

func Test__DeleteCheckRule__Setup(t *testing.T) {
	component := DeleteCheckRule{}

	t.Run("checkRuleId is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "checkRuleId is required")
	})

	t.Run("checkRuleId cannot be empty", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"checkRuleId": ""},
		})

		require.ErrorContains(t, err, "checkRuleId is required")
	})

	t.Run("dataset is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"checkRuleId": "test-rule-123"},
		})

		require.ErrorContains(t, err, "dataset is required")
	})

	t.Run("valid setup", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"checkRuleId": "test-rule-123",
				"dataset":     "default",
			},
		})

		require.NoError(t, err)
	})
}

func Test__DeleteCheckRule__Execute(t *testing.T) {
	component := DeleteCheckRule{}

	t.Run("successful deletion", func(t *testing.T) {
		execCtx := &contexts.ExecutionStateContext{}
		err := component.Execute(core.ExecutionContext{
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`
							{
								"success": true,
								"message": "Check rule deleted successfully"
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
				"dataset":     "default",
			},
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "default", execCtx.Channel)
		assert.Equal(t, "dash0.checkRule.deleted", execCtx.Type)
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
				"dataset":     "default",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete check rule")
	})
}
