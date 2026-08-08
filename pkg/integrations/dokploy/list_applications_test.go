package dokploy

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

func Test__Dokploy_ListApplications__Execute(t *testing.T) {
	component := &ListApplications{}

	t.Run("flattens applications across projects and environments", func(t *testing.T) {
		httpCtx := okResponse(projectsResponse)
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
			Configuration:  map[string]any{},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, ListApplicationsPayloadType, executionState.Type)
		require.Len(t, executionState.Payloads, 1)

		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, 2, data["count"])

		applications, ok := data["applications"].([]map[string]any)
		require.True(t, ok)
		require.Len(t, applications, 2)

		assert.Equal(t, "app-1", applications[0]["applicationId"])
		assert.Equal(t, "web", applications[0]["name"])
		assert.Equal(t, "done", applications[0]["status"])
		assert.Equal(t, "storefront", applications[0]["project"])
		assert.Equal(t, "production", applications[0]["environment"])
		assert.Equal(t, "api", applications[1]["name"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "/api/project.all", httpCtx.Requests[0].URL.Path)
	})

	t.Run("no projects -> empty list", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           okResponse(`[]`),
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
			Configuration:  map[string]any{},
		})

		require.NoError(t, err)
		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, 0, data["count"])

		applications, ok := data["applications"].([]map[string]any)
		require.True(t, ok)
		assert.Empty(t, applications)
	})

	t.Run("project with no applications -> empty list", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}

		err := component.Execute(core.ExecutionContext{
			HTTP:           okResponse(`[{"projectId":"p","name":"empty","environments":[{"environmentId":"e","name":"prod","applications":[]}]}]`),
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: executionState,
			Configuration:  map[string]any{},
		})

		require.NoError(t, err)
		data := readMap(readMap(executionState.Payloads[0])["data"])
		assert.Equal(t, 0, data["count"])
	})

	t.Run("API error -> surfaced", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"message":"Not authorized"}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    &contexts.IntegrationContext{Configuration: validIntegrationConfig()},
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  map[string]any{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Not authorized")
	})
}
