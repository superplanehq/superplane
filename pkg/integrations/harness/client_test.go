package harness

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__IsExecutionSummaryPipelineIdentifierFilterUnsupported(t *testing.T) {
	assert.True(t, IsExecutionSummaryPipelineIdentifierFilterUnsupported(
		&APIError{
			StatusCode: http.StatusBadRequest,
			Body:       `{"code":400,"message":"Unknown field pipelineIdentifier in request payload"}`,
		},
	))

	assert.True(t, IsExecutionSummaryPipelineIdentifierFilterUnsupported(
		&APIError{
			StatusCode: http.StatusUnprocessableEntity,
			Body:       `{"message":"cannot deserialize field pipelineIdentifier"}`,
		},
	))

	assert.False(t, IsExecutionSummaryPipelineIdentifierFilterUnsupported(
		&APIError{
			StatusCode: http.StatusBadRequest,
			Body:       `{"message":"invalid account"}`,
		},
	))
}

func Test__Client__ListExecutionSummariesPage__IncludesPipelineIdentifierFilter(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"data":{"content":[{"planExecutionId":"exec-1","pipelineIdentifier":"deploy","status":"SUCCESS","endTs":"1771266556263"}]}}`,
				)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken":  "pat.acc-123.test",
			"orgId":     "default",
			"projectId": "default_project",
		},
	}

	client, err := NewClient(httpCtx, integrationCtx)
	require.NoError(t, err)

	summaries, err := client.ListExecutionSummariesPage(0, 25, "deploy")
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "exec-1", summaries[0].ExecutionID)

	require.Len(t, httpCtx.Requests, 1)
	body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, readErr)
	assert.Contains(t, string(body), `"pipelineIdentifier":"deploy"`)
}

func Test__Client__ListExecutionSummariesPage__DoesNotFallbackOnFilterErrors(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Body: io.NopCloser(strings.NewReader(
					`{"code":400,"message":"Unknown field pipelineIdentifier in request payload"}`,
				)),
			},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken":  "pat.acc-123.test",
			"orgId":     "default",
			"projectId": "default_project",
		},
	}

	client, err := NewClient(httpCtx, integrationCtx)
	require.NoError(t, err)

	_, err = client.ListExecutionSummariesPage(0, 25, "deploy")
	require.Error(t, err)
	assert.True(t, IsExecutionSummaryPipelineIdentifierFilterUnsupported(err))
	require.Len(t, httpCtx.Requests, 1)
}

func Test__Client__ListOrganizations__ParsesTopLevelArray(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"identifier":"default","name":"Default Org"}]`,
				)),
			},
		},
	}

	client, err := NewClient(httpCtx, &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "pat.acc-123.test",
		},
	})
	require.NoError(t, err)

	organizations, err := client.ListOrganizations()
	require.NoError(t, err)
	require.Len(t, organizations, 1)
	assert.Equal(t, "default", organizations[0].Identifier)
	assert.Equal(t, "Default Org", organizations[0].Name)
}

func Test__Client__ListOrganizations__ParsesNestedHarnessShapes(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"status":"SUCCESS","data":{"content":[{"organization":{"identifier":"default","name":"default"}}]}}`,
				)),
			},
		},
	}

	client, err := NewClient(httpCtx, &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "pat.acc-123.test",
		},
	})
	require.NoError(t, err)

	organizations, err := client.ListOrganizations()
	require.NoError(t, err)
	require.Len(t, organizations, 1)
	assert.Equal(t, "default", organizations[0].Identifier)
	assert.Equal(t, "default", organizations[0].Name)
}

func Test__Client__ListProjects__ParsesTopLevelArray(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"identifier":"default_project","name":"Default Project"}]`,
				)),
			},
		},
	}

	client, err := NewClient(httpCtx, &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "pat.acc-123.test",
		},
	})
	require.NoError(t, err)

	projects, err := client.ListProjects("default")
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "default_project", projects[0].Identifier)
	assert.Equal(t, "Default Project", projects[0].Name)
}

func Test__Client__ListProjects__ParsesNestedHarnessShapes(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"status":"SUCCESS","data":{"content":[{"projectResponse":{"project":{"identifier":"default_project","name":"Default Project"}}}]}}`,
				)),
			},
		},
	}

	client, err := NewClient(httpCtx, &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "pat.acc-123.test",
		},
	})
	require.NoError(t, err)

	projects, err := client.ListProjects("default")
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "default_project", projects[0].Identifier)
	assert.Equal(t, "Default Project", projects[0].Name)
}

func Test__Client__ListOrganizations__ReturnsErrorWhenCallsFail(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"message":"invalid key"}`)),
			},
			{
				StatusCode: http.StatusUnauthorized,
				Body:       io.NopCloser(strings.NewReader(`{"message":"invalid key"}`)),
			},
		},
	}

	client, err := NewClient(httpCtx, &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiToken": "pat.acc-123.test",
		},
	})
	require.NoError(t, err)

	_, err = client.ListOrganizations()
	require.Error(t, err)
	require.ErrorContains(t, err, "request failed with 401")
}
