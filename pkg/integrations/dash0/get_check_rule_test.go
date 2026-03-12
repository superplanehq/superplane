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

func Test__GetCheckRule__Setup(t *testing.T) {
	component := GetCheckRule{}

	t.Run("checkRule is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "checkRule is required")
	})

	t.Run("checkRule cannot be empty", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"checkRule": ""},
		})

		require.ErrorContains(t, err, "checkRule is required")
	})

	t.Run("dataset is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration:   &contexts.IntegrationContext{},
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"checkRule": "test-rule-123"},
		})

		require.ErrorContains(t, err, "dataset is required")
	})

	t.Run("valid setup skips API when metadata already set", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Integration: &contexts.IntegrationContext{},
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{"checkRule": "test-rule-123", "checkRuleName": "Already Set"},
			},
			Configuration: map[string]any{
				"checkRule": "test-rule-123",
				"dataset":   "default",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid setup fetches check rule name from API", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`
							{
								"kind": "Dash0CheckRule",
								"metadata": {"name": "Test Alert"},
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
			Metadata: metadata,
			Configuration: map[string]any{
				"checkRule": "test-rule-123",
				"dataset":   "default",
			},
		})

		require.NoError(t, err)
		storedMetadata := metadata.Metadata.(CheckRuleNodeMetadata)
		assert.Equal(t, "Test Alert", storedMetadata.CheckRuleName)
	})
}

func Test__GetCheckRule__Execute(t *testing.T) {
	component := GetCheckRule{}

	t.Run("successful fetch", func(t *testing.T) {
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
				"checkRule": "test-rule-id-123",
				"dataset":   "default",
			},
		})

		require.NoError(t, err)
		assert.True(t, execCtx.Finished)
		assert.True(t, execCtx.Passed)
		assert.Equal(t, "default", execCtx.Channel)
		assert.Equal(t, "dash0.checkRule.fetched", execCtx.Type)
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
				"checkRule": "non-existent-rule",
				"dataset":   "default",
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get check rule")
	})
}
