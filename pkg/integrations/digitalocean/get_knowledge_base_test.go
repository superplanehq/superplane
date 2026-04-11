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

func Test__GetKnowledgeBase__Setup(t *testing.T) {
	component := &GetKnowledgeBase{}

	t.Run("missing knowledgeBase returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "knowledgeBase is required")
	})

	t.Run("empty knowledgeBase returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "knowledgeBase is required")
	})

	t.Run("expression knowledgeBase is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "{{ $.trigger.data.kbUUID }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid knowledgeBase -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"knowledgeBase": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"knowledge_base": {
								"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
								"name": "my-kb"
							}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "test-token",
				},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__GetKnowledgeBase__Execute(t *testing.T) {
	component := &GetKnowledgeBase{}

	kbResponse := `{
		"database_status": "ONLINE",
		"knowledge_base": {
			"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
			"name": "my-kb",
			"region": "tor1",
			"embedding_model_uuid": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
			"project_id": "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
			"database_id": "abf1055a-745d-4c24-a1db-1959ea819264",
			"tags": ["production", "docs"],
			"created_at": "2025-01-01T00:00:00Z",
			"updated_at": "2025-06-01T00:00:00Z",
			"last_indexing_job": {
				"uuid": "job-1",
				"status": "INDEX_JOB_STATUS_COMPLETED",
				"phase": "BATCH_JOB_PHASE_COMPLETE",
				"tokens": 123,
				"total_tokens": "12345",
				"completed_datasources": 2,
				"total_datasources": 2,
				"started_at": "2025-06-01T00:00:00Z",
				"finished_at": "2025-06-01T00:05:32Z",
				"created_at": "2025-06-01T00:00:00Z",
				"is_report_available": true,
				"knowledge_base_uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4"
			}
		}
	}`

	dataSourcesResponse := `{
		"knowledge_base_data_sources": [
			{
				"uuid": "ds-1",
				"bucket_name": "product-data",
				"region": "tor1",
				"chunking_algorithm": "CHUNKING_ALGORITHM_SECTION_BASED",
				"created_at": "2025-01-01T00:00:00Z",
				"updated_at": "2025-06-01T00:00:00Z"
			},
			{
				"uuid": "ds-2",
				"web_crawler_data_source": {
					"base_url": "https://docs.example.com",
					"crawling_option": "SCOPED",
					"embed_media": false
				},
				"chunking_algorithm": "CHUNKING_ALGORITHM_SEMANTIC",
				"created_at": "2025-02-01T00:00:00Z",
				"updated_at": "2025-06-01T00:00:00Z"
			}
		]
	}`

	modelsResponse := `{"models": [{"uuid": "05700391-7aa8-11ef-bf8f-4e013e2ddde4", "name": "GTE Large EN v1.5"}]}`
	projectsResponse := `{"projects": [{"id": "37455431-84bd-4fa2-94cf-e8486f8f8c5e", "name": "AI Agents"}]}`
	databasesResponse := `{"databases": [{"id": "abf1055a-745d-4c24-a1db-1959ea819264", "name": "my-kb-os", "engine": "opensearch", "status": "online"}]}`

	t.Run("successful fetch -> emits full KB data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(modelsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(projectsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(databasesResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dataSourcesResponse))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		require.Equal(t, "default", executionState.Channel)
		require.Equal(t, "digitalocean.knowledge_base.fetched", executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4", data["uuid"])
		assert.Equal(t, "my-kb", data["name"])
		assert.Equal(t, "ONLINE", data["databaseStatus"])
		assert.Equal(t, "tor1", data["region"])
		assert.Equal(t, "GTE Large EN v1.5", data["embeddingModelName"])
		assert.Equal(t, "AI Agents", data["projectName"])

		// Database
		db := data["database"].(map[string]any)
		assert.Equal(t, "abf1055a-745d-4c24-a1db-1959ea819264", db["id"])
		assert.Equal(t, "my-kb-os", db["name"])
		assert.Equal(t, "online", db["status"])

		// Data sources
		rawDS := data["dataSources"].([]map[string]any)
		assert.Len(t, rawDS, 2)

		// Spaces data source
		assert.Equal(t, "spaces", rawDS[0]["type"])
		assert.Equal(t, "tor1/product-data", rawDS[0]["spacesBucket"])
		assert.Equal(t, "2025-01-01T00:00:00Z", rawDS[0]["createdAt"])
		assert.Equal(t, "2025-06-01T00:00:00Z", rawDS[0]["updatedAt"])

		// Web data source
		assert.Equal(t, "web", rawDS[1]["type"])
		assert.Equal(t, "https://docs.example.com", rawDS[1]["webURL"])
		assert.Equal(t, "SCOPED", rawDS[1]["crawlingOption"])
		assert.Equal(t, "2025-02-01T00:00:00Z", rawDS[1]["createdAt"])

		// Last indexing job
		job := data["lastIndexingJob"].(map[string]any)
		assert.Equal(t, "INDEX_JOB_STATUS_COMPLETED", job["status"])
		assert.Equal(t, "BATCH_JOB_PHASE_COMPLETE", job["phase"])
		assert.Equal(t, "12345", job["totalTokens"])
		assert.Equal(t, 2, job["completedDataSources"])
		assert.Equal(t, 2, job["totalDataSources"])
		assert.Equal(t, true, job["isReportAvailable"])
	})

	t.Run("display name lookup failures do not block execution", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(kbResponse))},
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"forbidden"}`))},
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"forbidden"}`))},
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"forbidden"}`))},
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"forbidden"}`))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)

		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "my-kb", data["name"])
		assert.Nil(t, data["embeddingModelName"])
		assert.Nil(t, data["projectName"])
	})

	t.Run("KB without indexing job or database -> emits without those fields", func(t *testing.T) {
		minimalKBResponse := `{
			"database_status": "CREATING",
			"knowledge_base": {
				"uuid": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
				"name": "empty-kb",
				"region": "tor1",
				"embedding_model_uuid": "05700391-7aa8-11ef-bf8f-4e013e2ddde4",
				"project_id": "37455431-84bd-4fa2-94cf-e8486f8f8c5e",
				"tags": [],
				"created_at": "2025-01-01T00:00:00Z",
				"updated_at": "2025-01-01T00:00:00Z"
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(minimalKBResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(modelsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(projectsResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"knowledge_base_data_sources": []}`))},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase": "20cd8434-6ea1-11f0-bf8f-4e013e2ddde4",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		wrapped := executionState.Payloads[0].(map[string]any)
		data := wrapped["data"].(map[string]any)
		assert.Equal(t, "CREATING", data["databaseStatus"])
		assert.Nil(t, data["database"])
		assert.Nil(t, data["lastIndexingJob"])

		dataSources := data["dataSources"].([]map[string]any)
		assert.Empty(t, dataSources)
	})

	t.Run("non-existent KB -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"id":"not_found","message":"not found"}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"knowledgeBase": "non-existent-uuid",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get knowledge base")
		require.False(t, executionState.Passed)
	})
}

func Test__GetKnowledgeBase__Name(t *testing.T) {
	component := &GetKnowledgeBase{}
	require.Equal(t, "digitalocean.getKnowledgeBase", component.Name())
}

func Test__GetKnowledgeBase__Configuration(t *testing.T) {
	component := &GetKnowledgeBase{}
	fields := component.Configuration()

	require.Len(t, fields, 1)
	assert.Equal(t, "knowledgeBase", fields[0].Name)
	assert.Equal(t, "integration-resource", fields[0].Type)
	assert.True(t, fields[0].Required)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	assert.Equal(t, "knowledge_base", fields[0].TypeOptions.Resource.Type)
}
