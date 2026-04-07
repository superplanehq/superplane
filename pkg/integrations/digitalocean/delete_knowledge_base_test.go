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

func Test__DeleteKnowledgeBase__Setup(t *testing.T) {
	component := &DeleteKnowledgeBase{}

	t.Run("missing knowledgeBase returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "knowledgeBase is required")
	})

	t.Run("expression knowledgeBase is accepted at setup time", func(t *testing.T) {
		metaCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "{{ $.trigger.data.kbId }}",
			},
			Metadata: metaCtx,
		})

		require.NoError(t, err)
		meta, ok := metaCtx.Metadata.(DeleteKBNodeMetadata)
		require.True(t, ok)
		assert.Equal(t, "{{ $.trigger.data.kbId }}", meta.KnowledgeBaseID)
		assert.Equal(t, "{{ $.trigger.data.kbId }}", meta.KnowledgeBaseName)
	})

	t.Run("valid knowledgeBaseId -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "kb-uuid-123",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"knowledge_base": {"uuid": "kb-uuid-123", "name": "my-kb"}
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

func Test__DeleteKnowledgeBase__Execute(t *testing.T) {
	component := &DeleteKnowledgeBase{}

	t.Run("successful deletion without database -> emits confirmation", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase":            "kb-uuid-123",
				"deleteOpenSearchDatabase": false,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// DeleteKnowledgeBase
						StatusCode: http.StatusNoContent,
						Body:       io.NopCloser(strings.NewReader("")),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.knowledge_base.deleted", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped := executionState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "kb-uuid-123", payload["knowledgeBaseUUID"])
		assert.Equal(t, false, payload["databaseDeleted"])
	})

	t.Run("successful deletion with database -> emits confirmation with database info", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase":            "kb-uuid-123",
				"deleteOpenSearchDatabase": true,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// GetKnowledgeBase (to get database_id)
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"knowledge_base": {
								"uuid": "kb-uuid-123",
								"name": "my-kb",
								"database_id": "db-uuid-456"
							}
						}`)),
					},
					{
						// DeleteKnowledgeBase
						StatusCode: http.StatusNoContent,
						Body:       io.NopCloser(strings.NewReader("")),
					},
					{
						// ListDatabasesByEngine (resolve database name)
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"databases": [{"id": "db-uuid-456", "name": "my-kb-os", "engine": "opensearch"}]
						}`)),
					},
					{
						// DeleteDatabase
						StatusCode: http.StatusNoContent,
						Body:       io.NopCloser(strings.NewReader("")),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.knowledge_base.deleted", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped := executionState.Payloads[0].(map[string]any)
		payload := wrapped["data"].(map[string]any)
		assert.Equal(t, "kb-uuid-123", payload["knowledgeBaseUUID"])
		assert.Equal(t, true, payload["databaseDeleted"])
		assert.Equal(t, "db-uuid-456", payload["databaseId"])
		assert.Equal(t, "my-kb-os", payload["databaseName"])
	})

	t.Run("KB not found (404) -> emits success (idempotent) without database", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase":            "kb-uuid-123",
				"deleteOpenSearchDatabase": false,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(strings.NewReader(`{"message": "not found"}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.knowledge_base.deleted", executionState.Type)
	})

	t.Run("KB not found (404) with delete database flag -> emits success (idempotent)", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase":            "kb-uuid-123",
				"deleteOpenSearchDatabase": true,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// GetKnowledgeBase returns 404
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(strings.NewReader(`{"message": "not found"}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.knowledge_base.deleted", executionState.Type)
	})

	t.Run("API error (not 404) -> returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase":            "kb-uuid-123",
				"deleteOpenSearchDatabase": false,
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
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete knowledge base")
	})

	t.Run("KB deleted but database deletion fails -> returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase":            "kb-uuid-123",
				"deleteOpenSearchDatabase": true,
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// GetKnowledgeBase
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"knowledge_base": {"uuid": "kb-uuid-123", "name": "my-kb", "database_id": "db-uuid-456"}
						}`)),
					},
					{
						// DeleteKnowledgeBase
						StatusCode: http.StatusNoContent,
						Body:       io.NopCloser(strings.NewReader("")),
					},
					{
						// ListDatabasesByEngine (resolve database name)
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"databases": [{"id": "db-uuid-456", "name": "my-kb-os", "engine": "opensearch"}]
						}`)),
					},
					{
						// DeleteDatabase fails
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(strings.NewReader(`{"message": "internal error"}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete OpenSearch database")
	})
}
