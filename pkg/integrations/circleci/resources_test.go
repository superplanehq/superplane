package circleci

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

func Test__ListWorkflowNames(t *testing.T) {
	t.Run("missing project slug -> returns empty list", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := ListWorkflowNames(core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
			Parameters:  map[string]string{},
		})

		require.NoError(t, err)
		require.Empty(t, resources)
		require.Empty(t, httpContext.Requests)
	})

	t.Run("list workflows across pages and all branches", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [
							{"id": "build", "name": "build"},
							{"id": "deploy", "name": "deploy"}
						],
						"next_page_token": "next-page"
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"items": [
							{"id": "deploy", "name": "deploy"},
							{"id": "wf-id-only", "name": ""}
						],
						"next_page_token": ""
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		resources, err := ListWorkflowNames(core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
			Parameters: map[string]string{
				"projectSlug": "gh/acme/my-app",
			},
		})

		require.NoError(t, err)
		require.Len(t, resources, 3)
		assert.Equal(t, core.IntegrationResource{Type: ResourceTypeWorkflow, Name: "build", ID: "build"}, resources[0])
		assert.Equal(t, core.IntegrationResource{Type: ResourceTypeWorkflow, Name: "deploy", ID: "deploy"}, resources[1])
		assert.Equal(t, core.IntegrationResource{Type: ResourceTypeWorkflow, Name: "wf-id-only", ID: "wf-id-only"}, resources[2])

		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/insights/gh/acme/my-app/workflows?all-branches=true")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "all-branches=true")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "page-token=next-page")
	})
}
