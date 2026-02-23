package semaphore

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

func Test__Semaphore__ListResources(t *testing.T) {
	s := &Semaphore{}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"organizationUrl": "https://example.semaphoreci.com",
			"apiToken":        "token-123",
		},
	}

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		resources, err := s.ListResources("unknown", core.ListResourcesContext{})
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("pipeline resources by project id", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"name":"Build","ppl_id":"ppl-1","state":"done","running_at":"2026-02-23T10:00:00Z","done_at":"2026-02-23T10:09:00Z"},
						{"name":"Deploy","ppl_id":"ppl-2","state":"done","running_at":{"seconds":60,"nanos":0},"done_at":{"seconds":660,"nanos":0}}
					]`)),
				},
			},
		}

		resources, err := s.ListResources(ResourceTypePipeline, core.ListResourcesContext{
			Parameters:  map[string]string{"project_id": "project-1"},
			HTTP:        httpContext,
			Integration: integrationCtx,
		})
		require.NoError(t, err)
		require.Len(t, resources, 2)

		assert.Equal(t, ResourceTypePipeline, resources[0].Type)
		assert.Equal(t, "Build (done)", resources[0].Name)
		assert.Equal(t, "ppl-1", resources[0].ID)

		assert.Equal(t, ResourceTypePipeline, resources[1].Type)
		assert.Equal(t, "Deploy (done)", resources[1].Name)
		assert.Equal(t, "ppl-2", resources[1].ID)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(
			t,
			"https://example.semaphoreci.com/api/v1alpha/pipelines?project_id=project-1",
			httpContext.Requests[0].URL.String(),
		)
	})

	t.Run("pipeline resources across all projects", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"metadata":{"name":"Project A","id":"project-1"}},
						{"metadata":{"name":"Project B","id":"project-2"}}
					]`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`[{"name":"Build","ppl_id":"ppl-1","state":"done","running_at":"2026-02-23T10:00:00Z","done_at":"2026-02-23T10:09:00Z"}]`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"name":"Build","ppl_id":"ppl-1","state":"done","running_at":"2026-02-23T10:00:00Z","done_at":"2026-02-23T10:09:00Z"},
						{"name":"Test","ppl_id":"ppl-3","state":"done","running_at":"2026-02-23T10:00:00Z","done_at":"2026-02-23T10:11:00Z"}
					]`)),
				},
			},
		}

		resources, err := s.ListResources(ResourceTypePipeline, core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})
		require.NoError(t, err)
		require.Len(t, resources, 2)

		assert.Equal(t, "Build (done)", resources[0].Name)
		assert.Equal(t, "ppl-1", resources[0].ID)
		assert.Equal(t, "Test (done)", resources[1].Name)
		assert.Equal(t, "ppl-3", resources[1].ID)

		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/projects", httpContext.Requests[0].URL.String())
		assert.Equal(
			t,
			"https://example.semaphoreci.com/api/v1alpha/pipelines?project_id=project-1",
			httpContext.Requests[1].URL.String(),
		)
		assert.Equal(
			t,
			"https://example.semaphoreci.com/api/v1alpha/pipelines?project_id=project-2",
			httpContext.Requests[2].URL.String(),
		)
	})
}
