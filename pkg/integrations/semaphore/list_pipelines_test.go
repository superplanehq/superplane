package semaphore

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ListPipelines__Execute__Success(t *testing.T) {
	l := &ListPipelines{}

	t.Run("successfully lists pipelines", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{
							"ppl_id": "pipeline-1",
							"wf_id": "workflow-1",
							"name": "Pipeline 1",
							"state": "done",
							"result": "passed"
						},
						{
							"ppl_id": "pipeline-2",
							"wf_id": "workflow-2",
							"name": "Pipeline 2",
							"state": "done",
							"result": "failed"
						}
					]`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		spec := ListPipelinesSpec{
			Project: "test-project-id",
			Limit:   10,
		}

		metadata := ListPipelinesNodeMetadata{
			Project: &Project{
				ID:   "test-project-id",
				Name: "test-project",
				URL:  "https://example.semaphoreci.com/projects/test-project-id",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{
			Metadata: metadata,
		}

		stateCtx := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		executionCtx := core.ExecutionContext{
			Configuration:  spec,
			NodeMetadata:   nodeMetadataCtx,
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: stateCtx,
		}

		err := l.Execute(executionCtx)
		require.NoError(t, err)

		assert.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "project_id=test-project-id")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "limit=10")
		assert.NotEmpty(t, stateCtx.Payloads)
		assert.Equal(t, "default", stateCtx.Channel)
		assert.Equal(t, "semaphore.pipelines.listed", stateCtx.Type)

		payload, ok := stateCtx.Payloads[0].(map[string]any)
		require.True(t, ok)
		data, ok := payload["data"].(map[string]any)
		require.True(t, ok)
		pipelines, ok := data["pipelines"].([]any)
		require.True(t, ok)
		assert.Len(t, pipelines, 2)
	})

	t.Run("successfully lists pipelines with branch filter", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{
							"ppl_id": "pipeline-1",
							"wf_id": "workflow-1",
							"name": "Pipeline 1",
							"state": "done",
							"result": "passed"
						}
					]`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		spec := ListPipelinesSpec{
			Project:    "test-project-id",
			BranchName: "main",
		}

		metadata := ListPipelinesNodeMetadata{
			Project: &Project{
				ID:   "test-project-id",
				Name: "test-project",
				URL:  "https://example.semaphoreci.com/projects/test-project-id",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{
			Metadata: metadata,
		}

		stateCtx := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		executionCtx := core.ExecutionContext{
			Configuration:  spec,
			NodeMetadata:   nodeMetadataCtx,
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: stateCtx,
		}

		err := l.Execute(executionCtx)
		require.NoError(t, err)

		assert.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "project_id=test-project-id")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "branch_name=main")
	})

	t.Run("limits are enforced (max 100)", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		spec := ListPipelinesSpec{
			Project: "test-project-id",
			Limit:   200,
		}

		metadata := ListPipelinesNodeMetadata{
			Project: &Project{
				ID:   "test-project-id",
				Name: "test-project",
				URL:  "https://example.semaphoreci.com/projects/test-project-id",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{
			Metadata: metadata,
		}

		stateCtx := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		executionCtx := core.ExecutionContext{
			Configuration:  spec,
			NodeMetadata:   nodeMetadataCtx,
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: stateCtx,
		}

		err := l.Execute(executionCtx)
		require.NoError(t, err)

		assert.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "limit=100")
	})

	t.Run("uses default limit when not specified", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		spec := ListPipelinesSpec{
			Project: "test-project-id",
		}

		metadata := ListPipelinesNodeMetadata{
			Project: &Project{
				ID:   "test-project-id",
				Name: "test-project",
				URL:  "https://example.semaphoreci.com/projects/test-project-id",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{
			Metadata: metadata,
		}

		stateCtx := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		executionCtx := core.ExecutionContext{
			Configuration:  spec,
			NodeMetadata:   nodeMetadataCtx,
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: stateCtx,
		}

		err := l.Execute(executionCtx)
		require.NoError(t, err)

		assert.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "limit=30")
	})
}

func Test__ListPipelines__Execute__Error(t *testing.T) {
	l := &ListPipelines{}

	t.Run("error when API call fails", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("server error")),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		spec := ListPipelinesSpec{
			Project: "test-project-id",
		}

		metadata := ListPipelinesNodeMetadata{
			Project: &Project{
				ID:   "test-project-id",
				Name: "test-project",
				URL:  "https://example.semaphoreci.com/projects/test-project-id",
			},
		}

		nodeMetadataCtx := &contexts.MetadataContext{
			Metadata: metadata,
		}

		stateCtx := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		executionCtx := core.ExecutionContext{
			Configuration:  spec,
			NodeMetadata:   nodeMetadataCtx,
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: stateCtx,
		}

		err := l.Execute(executionCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing pipelines")
	})
}

func Test__ListPipelines__Setup(t *testing.T) {
	l := &ListPipelines{}

	t.Run("setup stores project metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"metadata": {
							"name": "test-project",
							"id": "test-project-id"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		spec := ListPipelinesSpec{
			Project: "test-project",
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{},
		}

		setupCtx := core.SetupContext{
			Configuration: spec,
			Metadata:      metadataCtx,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		}

		err := l.Setup(setupCtx)
		require.NoError(t, err)

		metadata := metadataCtx.Metadata.(ListPipelinesNodeMetadata)
		assert.Equal(t, "test-project-id", metadata.Project.ID)
		assert.Equal(t, "test-project", metadata.Project.Name)
	})
}

func Test__ListPipelinesClient__ListPipelinesWithFilters(t *testing.T) {
	t.Run("builds correct query parameters", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{
							"ppl_id": "pipeline-1",
							"wf_id": "workflow-1",
							"name": "Pipeline 1",
							"state": "done",
							"result": "passed"
						}
					]`)),
				},
			},
		}

		client := &Client{
			OrgURL:   "https://example.semaphoreci.com",
			APIToken: "token-123",
			http:     httpContext,
		}

		params := &ListPipelinesParams{
			ProjectID:     "test-project",
			BranchName:    "main",
			YMLFilePath:   ".semaphore/semaphore.yml",
			CreatedAfter:  "2023-01-01T00:00:00Z",
			CreatedBefore: "2024-01-01T00:00:00Z",
			DoneAfter:     "2023-06-01T00:00:00Z",
			DoneBefore:    "2023-12-31T23:59:59Z",
			Limit:         50,
		}

		pipelines, err := client.ListPipelinesWithFilters(params)
		require.NoError(t, err)
		assert.Len(t, pipelines, 1)

		assert.Len(t, httpContext.Requests, 1)
		requestURL := httpContext.Requests[0].URL.String()
		assert.Contains(t, requestURL, "project_id=test-project")
		assert.Contains(t, requestURL, "branch_name=main")
		assert.Contains(t, requestURL, "yml_file_path="+url.QueryEscape(".semaphore/semaphore.yml"))
		assert.Contains(t, requestURL, "created_after="+url.QueryEscape("2023-01-01T00:00:00Z"))
		assert.Contains(t, requestURL, "created_before="+url.QueryEscape("2024-01-01T00:00:00Z"))
		assert.Contains(t, requestURL, "done_after="+url.QueryEscape("2023-06-01T00:00:00Z"))
		assert.Contains(t, requestURL, "done_before="+url.QueryEscape("2023-12-31T23:59:59Z"))
		assert.Contains(t, requestURL, "limit=50")
	})

	t.Run("follows pagination links", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Header: http.Header{
						"Link": []string{
							"<https://example.semaphoreci.com/api/v1alpha/pipelines?page=2&project_id=test-project&limit=2>; rel=\"next\"",
						},
					},
					Body: io.NopCloser(strings.NewReader(`[
						{"ppl_id": "pipeline-1"},
						{"ppl_id": "pipeline-2"}
					]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"ppl_id": "pipeline-3"},
						{"ppl_id": "pipeline-4"}
					]`)),
				},
			},
		}

		client := &Client{
			OrgURL:   "https://example.semaphoreci.com",
			APIToken: "token-123",
			http:     httpContext,
		}

		params := &ListPipelinesParams{
			ProjectID: "test-project",
			Limit:     3,
		}

		pipelines, err := client.ListPipelinesWithFilters(params)
		require.NoError(t, err)
		assert.Len(t, pipelines, 3)
		assert.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "page=2")
	})

	t.Run("enforces limit max of 100", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		client := &Client{
			OrgURL:   "https://example.semaphoreci.com",
			APIToken: "token-123",
			http:     httpContext,
		}

		params := &ListPipelinesParams{
			ProjectID: "test-project",
			Limit:     200,
		}

		_, err := client.ListPipelinesWithFilters(params)
		require.NoError(t, err)

		assert.Contains(t, httpContext.Requests[0].URL.String(), "limit=100")
	})

	t.Run("uses default limit of 30 when not specified", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		client := &Client{
			OrgURL:   "https://example.semaphoreci.com",
			APIToken: "token-123",
			http:     httpContext,
		}

		params := &ListPipelinesParams{
			ProjectID: "test-project",
		}

		_, err := client.ListPipelinesWithFilters(params)
		require.NoError(t, err)

		assert.Contains(t, httpContext.Requests[0].URL.String(), "limit=30")
	})

	t.Run("returns error when project_id is missing", func(t *testing.T) {
		client := &Client{
			OrgURL:   "https://example.semaphoreci.com",
			APIToken: "token-123",
		}

		params := &ListPipelinesParams{}

		_, err := client.ListPipelinesWithFilters(params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project_id is required")
	})
}
