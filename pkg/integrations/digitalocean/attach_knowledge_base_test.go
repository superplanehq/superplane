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

func Test__AttachKnowledgeBase__Setup(t *testing.T) {
	component := &AttachKnowledgeBase{}

	t.Run("missing agent returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "agent is required")
	})

	t.Run("missing knowledgeBase returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agent": "agent-uuid",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "knowledgeBase is required")
	})

	t.Run("expression agent is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agent":         "{{ $.trigger.data.agentId }}",
				"knowledgeBase": "kb-uuid",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// GetKnowledgeBase (literal KB id still resolved for display)
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"knowledge_base": {"uuid": "kb-uuid", "name": "my-kb"}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("expression knowledgeBase is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agent":         "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"knowledgeBase": "{{ $.steps.create_kb.id }}",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// GetAgent (literal agent id still resolved for display)
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"agent": {"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", "name": "my-agent"}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agent":         "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"knowledgeBase": "kb-uuid",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// GetAgent
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"agent": {"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", "name": "my-agent"}
						}`)),
					},
					{
						// GetKnowledgeBase
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"knowledge_base": {"uuid": "kb-uuid", "name": "my-kb"}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__AttachKnowledgeBase__Execute(t *testing.T) {
	component := &AttachKnowledgeBase{}

	t.Run("successful attach -> emits confirmation", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"agent":         "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"knowledgeBase": "kb-uuid",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// AttachKnowledgeBase
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.knowledge_base.attached", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped := executionState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", payload["agentUUID"])
		assert.Equal(t, "kb-uuid", payload["knowledgeBaseUUID"])
	})

	t.Run("attach API error -> returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"agent":         "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"knowledgeBase": "kb-uuid",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusUnprocessableEntity,
						Body:       io.NopCloser(strings.NewReader(`{"message": "knowledge base not found"}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to attach knowledge base")
	})
}
