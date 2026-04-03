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

func Test__UpdateAgent__Setup(t *testing.T) {
	component := &UpdateAgent{}

	t.Run("missing agentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "agentId is required")
	})

	t.Run("missing oldKnowledgeBaseId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agentId": "agent-uuid",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "oldKnowledgeBaseId is required")
	})

	t.Run("missing newKnowledgeBaseId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agentId":            "agent-uuid",
				"oldKnowledgeBaseId": "kb-v1-uuid",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "newKnowledgeBaseId is required")
	})

	t.Run("same old and new knowledge base ID returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agentId":            "agent-uuid",
				"oldKnowledgeBaseId": "kb-uuid",
				"newKnowledgeBaseId": "kb-uuid",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "oldKnowledgeBaseId and newKnowledgeBaseId must be different")
	})

	t.Run("expression agentId is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agentId":            "{{ $.trigger.data.agentId }}",
				"oldKnowledgeBaseId": "kb-v1-uuid",
				"newKnowledgeBaseId": "kb-v2-uuid",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agentId":            "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"oldKnowledgeBaseId": "kb-v1-uuid",
				"newKnowledgeBaseId": "kb-v2-uuid",
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
						// GetKnowledgeBase (old KB)
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"knowledge_base": {"uuid": "kb-v1-uuid", "name": "docs-kb-v1"}
						}`)),
					},
					{
						// GetKnowledgeBase (new KB)
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"knowledge_base": {"uuid": "kb-v2-uuid", "name": "docs-kb-v2"}
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

func Test__UpdateAgent__Execute(t *testing.T) {
	component := &UpdateAgent{}

	t.Run("successful swap -> emits confirmation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					// DetachKnowledgeBase
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
				{
					// AttachKnowledgeBase
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"agentId":            "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"oldKnowledgeBaseId": "kb-v1-uuid",
				"newKnowledgeBaseId": "kb-v2-uuid",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.agent.updated", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped := executionState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", payload["agentId"])
		assert.Equal(t, "kb-v1-uuid", payload["previousKnowledgeBaseId"])
		assert.Equal(t, "kb-v2-uuid", payload["newKnowledgeBaseId"])
	})

	t.Run("detach returns 404 (already detached) -> proceeds with attach", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					// DetachKnowledgeBase - already detached
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "not found"}`)),
				},
				{
					// AttachKnowledgeBase
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"agentId":            "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"oldKnowledgeBaseId": "kb-v1-uuid",
				"newKnowledgeBaseId": "kb-v2-uuid",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.agent.updated", executionState.Type)
	})

	t.Run("detach API error (non-404) -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"message": "internal error"}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"agentId":            "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"oldKnowledgeBaseId": "kb-v1-uuid",
				"newKnowledgeBaseId": "kb-v2-uuid",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to detach knowledge base")
	})

	t.Run("attach API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					// DetachKnowledgeBase succeeds
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
				{
					// AttachKnowledgeBase fails
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"message": "knowledge base not found"}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"agentId":            "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"oldKnowledgeBaseId": "kb-v1-uuid",
				"newKnowledgeBaseId": "kb-v2-uuid",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to attach knowledge base")
	})
}
