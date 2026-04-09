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

func Test__IndexKnowledgeBase__Setup(t *testing.T) {
	component := &IndexKnowledgeBase{}

	t.Run("missing knowledgeBase returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "knowledgeBase is required")
	})

	t.Run("expression knowledgeBase is accepted", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "{{ $.trigger.data.kbUUID }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid knowledgeBase resolves metadata", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "kb-uuid-1",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"database_status": "ONLINE",
							"knowledge_base": {
								"uuid": "kb-uuid-1",
								"name": "my-kb"
							}
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

func Test__IndexKnowledgeBase__Execute(t *testing.T) {
	component := &IndexKnowledgeBase{}

	kbResponse := `{
		"database_status": "ONLINE",
		"knowledge_base": {
			"uuid": "kb-uuid-1",
			"name": "my-kb",
			"region": "tor1"
		}
	}`

	startJobResponse := `{
		"job": {
			"uuid": "job-uuid-1",
			"status": "INDEX_JOB_STATUS_PENDING",
			"knowledge_base_uuid": "kb-uuid-1"
		}
	}`

	t.Run("starts indexing job and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(startJobResponse))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase": "kb-uuid-1",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       metadata,
			Requests:       requests,
		})

		require.NoError(t, err)
		require.False(t, executionState.Passed)
		require.Equal(t, "poll", requests.Action)

		stored, ok := metadata.Metadata.(indexKBMetadata)
		require.True(t, ok)
		require.Equal(t, "job-uuid-1", stored.JobID)
	})

	t.Run("returns error when start job fails", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbResponse))},
				{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"message":"error"}`))},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase": "kb-uuid-1",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to start indexing job")
	})
}

func Test__IndexKnowledgeBase__HandleAction(t *testing.T) {
	component := &IndexKnowledgeBase{}

	meta := map[string]any{
		"kbUUID": "kb-uuid-1",
		"kbName": "my-kb",
		"jobId":  "job-uuid-1",
	}

	t.Run("completed job emits output", func(t *testing.T) {
		jobCompleted := `{
			"job": {
				"uuid": "job-uuid-1",
				"status": "INDEX_JOB_STATUS_COMPLETED",
				"phase": "BATCH_JOB_PHASE_SUCCEEDED",
				"tokens": 500,
				"total_tokens": "1500",
				"completed_datasources": 2,
				"total_datasources": 2,
				"started_at": "2025-06-01T00:00:00Z",
				"finished_at": "2025-06-01T00:05:32Z",
				"is_report_available": true,
				"knowledge_base_uuid": "kb-uuid-1"
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(jobCompleted))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{Metadata: meta},
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, "default", executionState.Channel)
		require.Equal(t, "digitalocean.knowledge_base.indexed", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "kb-uuid-1", data["knowledgeBaseUUID"])
		assert.Equal(t, "my-kb", data["knowledgeBaseName"])
		assert.Equal(t, "job-uuid-1", data["jobUUID"])
		assert.Equal(t, "INDEX_JOB_STATUS_COMPLETED", data["status"])
		assert.Equal(t, "BATCH_JOB_PHASE_SUCCEEDED", data["phase"])
		assert.Equal(t, "1500", data["totalTokens"])
		assert.Equal(t, 2, data["completedDataSources"])
		assert.Equal(t, 2, data["totalDataSources"])
		assert.Equal(t, true, data["isReportAvailable"])
	})

	t.Run("running job reschedules poll", func(t *testing.T) {
		jobRunning := `{
			"job": {
				"uuid": "job-uuid-1",
				"status": "INDEX_JOB_STATUS_RUNNING",
				"phase": "BATCH_JOB_PHASE_RUNNING",
				"completed_datasources": 1,
				"total_datasources": 2,
				"knowledge_base_uuid": "kb-uuid-1"
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(jobRunning))},
			},
		}

		requests := &contexts.RequestContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
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
		jobFailed := `{
			"job": {
				"uuid": "job-uuid-1",
				"status": "INDEX_JOB_STATUS_FAILED",
				"knowledge_base_uuid": "kb-uuid-1"
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(jobFailed))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{Metadata: meta},
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.False(t, executionState.Passed)
		require.Contains(t, executionState.FailureMessage, "INDEX_JOB_STATUS_FAILED")
	})

	t.Run("job not found yet reschedules poll", func(t *testing.T) {
		notFound := `{"message":"not found"}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(notFound))},
			},
		}

		requests := &contexts.RequestContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
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

	t.Run("mismatched kb uuid fails execution", func(t *testing.T) {
		jobWrongKB := `{
			"job": {
				"uuid": "job-uuid-1",
				"status": "INDEX_JOB_STATUS_COMPLETED",
				"knowledge_base_uuid": "kb-uuid-other"
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(jobWrongKB))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{Metadata: meta},
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		require.False(t, executionState.Passed)
		require.Contains(t, executionState.FailureMessage, "belongs to knowledge base")
	})
}

func Test__IndexKnowledgeBase__Name(t *testing.T) {
	component := &IndexKnowledgeBase{}
	require.Equal(t, "digitalocean.indexKnowledgeBase", component.Name())
}

func Test__IndexKnowledgeBase__Configuration(t *testing.T) {
	component := &IndexKnowledgeBase{}
	fields := component.Configuration()

	require.Len(t, fields, 1)
	assert.Equal(t, "knowledgeBase", fields[0].Name)
	assert.True(t, fields[0].Required)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	assert.Equal(t, "knowledge_base", fields[0].TypeOptions.Resource.Type)
}
