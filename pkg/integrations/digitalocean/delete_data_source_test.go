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

func Test__DeleteDataSource__Setup(t *testing.T) {
	component := &DeleteDataSource{}

	t.Run("missing knowledgeBase returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"dataSource": "ds-uuid-1",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "knowledgeBase is required")
	})

	t.Run("missing dataSource returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "{{ $.trigger.data.kbUUID }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "dataSource is required")
	})

	t.Run("valid configuration with expressions is accepted", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "{{ $.trigger.data.kbUUID }}",
				"dataSource":    "{{ $.trigger.data.dsUUID }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration with UUIDs resolves metadata", func(t *testing.T) {
		kbResponse := `{
			"database_status": "ONLINE",
			"knowledge_base": {"uuid": "kb-uuid-1", "name": "my-kb"}
		}`

		dsListResponse := `{
			"knowledge_base_data_sources": [
				{"uuid": "ds-uuid-1", "bucket_name": "my-bucket", "region": "tor1"}
			]
		}`

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "kb-uuid-1",
				"dataSource":    "ds-uuid-1",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbResponse))},
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dsListResponse))},
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

func Test__DeleteDataSource__Execute(t *testing.T) {
	component := &DeleteDataSource{}

	kbResponse := `{
		"database_status": "ONLINE",
		"knowledge_base": {"uuid": "kb-uuid-1", "name": "my-kb", "region": "tor1"}
	}`

	deleteResponse := `{}`

	t.Run("deletes data source and polls for auto-triggered re-indexing", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbResponse))},
				{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader(deleteResponse))},
			},
		}

		requests := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase": "kb-uuid-1",
				"dataSource":    "ds-uuid-1",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       requests,
		})

		require.NoError(t, err)
		require.False(t, executionState.Passed)
		require.Equal(t, "poll", requests.Action)
	})
}

func Test__DeleteDataSource__HandleAction(t *testing.T) {
	component := &DeleteDataSource{}

	meta := map[string]any{
		"kbUUID":         "kb-uuid-1",
		"kbName":         "my-kb",
		"dataSourceUUID": "ds-uuid-1",
		"output": map[string]any{
			"dataSourceUUID":    "ds-uuid-1",
			"knowledgeBaseUUID": "kb-uuid-1",
			"knowledgeBaseName": "my-kb",
		},
	}

	t.Run("completed job emits output with indexing details", func(t *testing.T) {
		kbWithCompletedJob := `{
			"database_status": "ONLINE",
			"knowledge_base": {
				"uuid": "kb-uuid-1",
				"name": "my-kb",
				"last_indexing_job": {
					"uuid": "job-uuid-1",
					"status": "INDEX_JOB_STATUS_COMPLETED",
					"total_tokens": "800",
					"completed_datasources": 1,
					"total_datasources": 1,
					"started_at": "2025-06-01T00:00:00Z",
					"finished_at": "2025-06-01T00:03:12Z"
				}
			}
		}`

		executionState := &contexts.ExecutionStateContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbWithCompletedJob))},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{Metadata: meta},
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, "digitalocean.data_source.deleted", executionState.Type)

		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "ds-uuid-1", data["dataSourceUUID"])

		job := data["indexingJob"].(map[string]any)
		assert.Equal(t, "INDEX_JOB_STATUS_COMPLETED", job["status"])
	})

	t.Run("running job reschedules poll", func(t *testing.T) {
		kbWithRunningJob := `{
			"database_status": "ONLINE",
			"knowledge_base": {
				"uuid": "kb-uuid-1",
				"name": "my-kb",
				"last_indexing_job": {
					"uuid": "job-uuid-1",
					"status": "INDEX_JOB_STATUS_RUNNING"
				}
			}
		}`

		requests := &contexts.RequestContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbWithRunningJob))},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       &contexts.MetadataContext{Metadata: meta},
			Requests:       requests,
		})

		require.NoError(t, err)
		require.Equal(t, "poll", requests.Action)
	})

	t.Run("failed job fails execution", func(t *testing.T) {
		kbWithFailedJob := `{
			"database_status": "ONLINE",
			"knowledge_base": {
				"uuid": "kb-uuid-1",
				"name": "my-kb",
				"last_indexing_job": {
					"uuid": "job-uuid-1",
					"status": "INDEX_JOB_STATUS_FAILED"
				}
			}
		}`

		executionState := &contexts.ExecutionStateContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbWithFailedJob))},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{Metadata: meta},
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.True(t, executionState.Finished)
		require.False(t, executionState.Passed)
		require.Contains(t, executionState.FailureMessage, "INDEX_JOB_STATUS_FAILED")
	})

	t.Run("partial job fails execution", func(t *testing.T) {
		kbWithPartialJob := `{
			"database_status": "ONLINE",
			"knowledge_base": {
				"uuid": "kb-uuid-1",
				"name": "my-kb",
				"last_indexing_job": {
					"uuid": "job-uuid-1",
					"status": "INDEX_JOB_STATUS_PARTIAL"
				}
			}
		}`

		executionState := &contexts.ExecutionStateContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbWithPartialJob))},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{Metadata: meta},
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.True(t, executionState.Finished)
		require.False(t, executionState.Passed)
		require.Contains(t, executionState.FailureMessage, "INDEX_JOB_STATUS_PARTIAL")
	})
}

func Test__DeleteDataSource__Name(t *testing.T) {
	component := &DeleteDataSource{}
	require.Equal(t, "digitalocean.deleteDataSource", component.Name())
}

func Test__DeleteDataSource__Configuration(t *testing.T) {
	component := &DeleteDataSource{}
	fields := component.Configuration()

	require.Len(t, fields, 2)
	assert.Equal(t, "knowledgeBase", fields[0].Name)
	assert.True(t, fields[0].Required)
	assert.Equal(t, "dataSource", fields[1].Name)
	assert.True(t, fields[1].Required)
}
