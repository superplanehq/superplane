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

func Test__DetachKnowledgeBase__Setup(t *testing.T) {
	component := &DetachKnowledgeBase{}

	t.Run("missing agentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "agentId is required")
	})

	t.Run("missing knowledgeBaseId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agentId": "agent-uuid",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "knowledgeBaseId is required")
	})

	t.Run("expression agentId is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agentId":         "{{ $.trigger.data.agentId }}",
				"knowledgeBaseId": "kb-uuid",
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

	t.Run("expression knowledgeBaseId is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agentId":         "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"knowledgeBaseId": "{{ $.steps.create_kb.id }}",
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
				"agentId":         "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"knowledgeBaseId": "kb-uuid",
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

func Test__DetachKnowledgeBase__Execute(t *testing.T) {
	component := &DetachKnowledgeBase{}

	t.Run("successful detach -> emits confirmation", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"agentId":         "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"knowledgeBaseId": "kb-uuid",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// DetachKnowledgeBase
						StatusCode: http.StatusNoContent,
						Body:       io.NopCloser(strings.NewReader("")),
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
		assert.Equal(t, "digitalocean.knowledge_base.detached", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped := executionState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", payload["agentId"])
		assert.Equal(t, "kb-uuid", payload["knowledgeBaseId"])
	})

	t.Run("detach API error -> returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"agentId":         "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"knowledgeBaseId": "kb-uuid",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(strings.NewReader(`{"message": "internal error"}`)),
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
		assert.Contains(t, err.Error(), "failed to detach knowledge base")
	})
}
