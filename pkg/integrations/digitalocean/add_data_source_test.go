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

func Test__AddDataSource__Setup(t *testing.T) {
	component := &AddDataSource{}

	t.Run("missing knowledgeBase returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"type":         "spaces",
				"spacesBucket": "tor1/my-bucket",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "knowledgeBase is required")
	})

	t.Run("missing data source type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "{{ $.trigger.data.kbUUID }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "type is required")
	})

	t.Run("valid spaces data source is accepted", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "kb-uuid-1",
				"type":          "spaces",
				"spacesBucket":  "tor1/my-bucket",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"database_status": "ONLINE",
							"knowledge_base": {"uuid": "kb-uuid-1", "name": "my-kb"}
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

	t.Run("valid web data source is accepted", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase":  "{{ $.trigger.data.kbUUID }}",
				"type":           "web",
				"crawlType":      "seed",
				"webURL":         "https://docs.example.com",
				"crawlingOption": "SCOPED",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__AddDataSource__Execute(t *testing.T) {
	component := &AddDataSource{}

	kbResponse := `{
		"database_status": "ONLINE",
		"knowledge_base": {"uuid": "kb-uuid-1", "name": "my-kb", "region": "tor1"}
	}`

	addDSResponse := `{
		"knowledge_base_data_source": {
			"uuid": "ds-uuid-1",
			"bucket_name": "my-bucket",
			"region": "tor1",
			"chunking_algorithm": "CHUNKING_ALGORITHM_SECTION_BASED"
		}
	}`

	startJobResponse := `{
		"job": {
			"uuid": "job-uuid-1",
			"status": "INDEX_JOB_STATUS_PENDING",
			"knowledge_base_uuid": "kb-uuid-1"
		}
	}`

	t.Run("adds data source and starts indexing for only the new data source when indexAfterAdding is true", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(addDSResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(startJobResponse))},
			},
		}

		requests := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase":    "kb-uuid-1",
				"type":             "spaces",
				"spacesBucket":     "tor1/my-bucket",
				"indexAfterAdding": true,
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

	t.Run("adds data source and emits immediately when indexAfterAdding is false", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(addDSResponse))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase":    "kb-uuid-1",
				"type":             "spaces",
				"spacesBucket":     "tor1/my-bucket",
				"indexAfterAdding": false,
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, "default", executionState.Channel)
		require.Equal(t, "digitalocean.data_source.added", executionState.Type)

		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "ds-uuid-1", data["dataSourceUUID"])
		assert.Equal(t, "kb-uuid-1", data["knowledgeBaseUUID"])
		assert.Equal(t, "my-kb", data["knowledgeBaseName"])
	})
}

func Test__AddDataSource__HandleAction(t *testing.T) {
	component := &AddDataSource{}

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
					"total_tokens": "500",
					"completed_datasources": 1,
					"total_datasources": 1,
					"started_at": "2025-06-01T00:00:00Z",
					"finished_at": "2025-06-01T00:05:32Z"
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

		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "ds-uuid-1", data["dataSourceUUID"])
		assert.Equal(t, "kb-uuid-1", data["knowledgeBaseUUID"])

		job := data["indexingJob"].(map[string]any)
		assert.Equal(t, "INDEX_JOB_STATUS_COMPLETED", job["status"])
		assert.Equal(t, "500", job["totalTokens"])
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
}

func Test__AddDataSource__Name(t *testing.T) {
	component := &AddDataSource{}
	require.Equal(t, "digitalocean.addDataSource", component.Name())
}

func Test__AddDataSource__Configuration(t *testing.T) {
	component := &AddDataSource{}
	fields := component.Configuration()

	// First two are knowledgeBase and indexAfterAdding, rest are from dataSourceItemSchema
	require.True(t, len(fields) > 2)
	assert.Equal(t, "knowledgeBase", fields[0].Name)
	assert.True(t, fields[0].Required)
	assert.Equal(t, "indexAfterAdding", fields[1].Name)
}
