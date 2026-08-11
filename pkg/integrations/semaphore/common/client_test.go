package common

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func newTestClient(httpContext *contexts.HTTPContext) *Client {
	return &Client{
		OrgURL:   "https://example.semaphoreci.com",
		APIToken: "token-123",
		http:     httpContext,
	}
}

func Test__Client__ListPipelines(t *testing.T) {
	t.Run("without branch filter -> does not add query param", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		client := newTestClient(httpContext)
		pipelines, err := client.ListPipelines("proj-123", "")
		require.NoError(t, err)
		assert.Empty(t, pipelines)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/pipelines?project_id=proj-123", httpContext.Requests[0].URL.String())
	})

	t.Run("with branch filter -> adds branch_name query param", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"ppl_id": "ppl-1", "branch_name": "main", "commit_sha": "abc123"}
					]`)),
				},
			},
		}

		client := newTestClient(httpContext)
		pipelines, err := client.ListPipelines("proj-123", "main")
		require.NoError(t, err)
		require.Len(t, pipelines, 1)
		assert.Equal(t, "ppl-1", pipelines[0].PipelineID)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/pipelines?project_id=proj-123&branch_name=main", httpContext.Requests[0].URL.String())
	})

	t.Run("HTTP error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error": "boom"}`)),
				},
			},
		}

		client := newTestClient(httpContext)
		_, err := client.ListPipelines("proj-123", "")
		require.Error(t, err)
	})
}

func Test__Client__FindPipelineByBranchAndCommit(t *testing.T) {
	t.Run("matching pipeline found", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"ppl_id": "ppl-1", "branch_name": "main", "commit_sha": "other"},
						{"ppl_id": "ppl-2", "branch_name": "main", "commit_sha": "abc123"}
					]`)),
				},
			},
		}

		client := newTestClient(httpContext)
		pipeline, err := client.FindPipelineByBranchAndCommit("proj-123", "main", "abc123")
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		assert.Equal(t, "ppl-2", pipeline.PipelineID)
	})

	t.Run("no matching pipeline -> ErrPipelineNotFound", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"ppl_id": "ppl-1", "branch_name": "main", "commit_sha": "other"}
					]`)),
				},
			},
		}

		client := newTestClient(httpContext)
		pipeline, err := client.FindPipelineByBranchAndCommit("proj-123", "main", "abc123")
		require.Nil(t, pipeline)
		require.Error(t, err)
		assert.True(t, IsNotFoundError(err))
	})

	t.Run("HTTP error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error": "boom"}`)),
				},
			},
		}

		client := newTestClient(httpContext)
		pipeline, err := client.FindPipelineByBranchAndCommit("proj-123", "main", "abc123")
		require.Nil(t, pipeline)
		require.Error(t, err)
		assert.False(t, IsNotFoundError(err))
	})
}
