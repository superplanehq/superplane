package gitlab

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func ciMinutesNamespaceUsageResponseBody() string {
	return `{
		"data": {
			"ciMinutesUsage": {
				"nodes": [{"month": "July", "monthIso8601": "2026-07-01", "minutes": 245, "sharedRunnersDuration": 14700}]
			}
		}
	}`
}

func ciMinutesProjectUsageResponseBody() string {
	return `{
		"data": {
			"ciMinutesProjectMonthlyUsage": {
				"nodes": [
					{"minutes": 180, "sharedRunnersDuration": 10800, "project": {"id": "gid://gitlab/Project/1", "name": "hello-world", "fullPath": "felixgateru/hello-world"}},
					{"minutes": 65, "sharedRunnersDuration": 3900, "project": {"id": "gid://gitlab/Project/2", "name": "other-project", "fullPath": "felixgateru/other-project"}}
				],
				"pageInfo": {"hasNextPage": false, "endCursor": ""}
			}
		}
	}`
}

func Test__GetPipelineMinutesUsage__Execute(t *testing.T) {
	c := &GetPipelineMinutesUsage{}

	t.Run("success - personal namespace", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, ciMinutesNamespaceUsageResponseBody()),
					GitlabMockResponse(http.StatusOK, ciMinutesProjectUsageResponseBody()),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		assert.Equal(t, "gitlab.pipelineMinutesUsage", executionState.Type)

		var result PipelineMinutesUsageResult
		payloadBytes, _ := json.Marshal(payload["data"])
		json.Unmarshal(payloadBytes, &result)

		assert.Equal(t, "July", result.Month)
		assert.Equal(t, 245, result.Minutes)
		assert.Equal(t, 14700, result.SharedRunnersDuration)
		require.Len(t, result.Projects, 2)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 2)
		for _, req := range httpCtx.Requests {
			assert.Equal(t, "https://gitlab.com/api/graphql", req.URL.String())
		}
	})

	t.Run("success - resolves group to a namespace GID first", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
					"groupId":     "my-org",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{"id": 456, "full_path": "my-org"}`),
					GitlabMockResponse(http.StatusOK, ciMinutesNamespaceUsageResponseBody()),
					GitlabMockResponse(http.StatusOK, ciMinutesProjectUsageResponseBody()),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 3, "a root group must not trigger a second group lookup")
		assert.Equal(t, "https://gitlab.com/api/v4/groups/my-org", httpCtx.Requests[0].URL.String())

		for _, req := range httpCtx.Requests[1:] {
			body, _ := io.ReadAll(req.Body)
			var reqBody map[string]any
			json.Unmarshal(body, &reqBody)
			variables := reqBody["variables"].(map[string]any)
			assert.Equal(t, "gid://gitlab/Group/456", variables["namespaceId"])
		}
	})

	t.Run("success - resolves a subgroup to its root group, since minutes are only tracked there", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
					"groupId":     "my-org/my-subgroup",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{"id": 789, "full_path": "my-org/my-subgroup"}`),
					GitlabMockResponse(http.StatusOK, `{"id": 456, "full_path": "my-org"}`),
					GitlabMockResponse(http.StatusOK, ciMinutesNamespaceUsageResponseBody()),
					GitlabMockResponse(http.StatusOK, ciMinutesProjectUsageResponseBody()),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 4)
		assert.Equal(t, "https://gitlab.com/api/v4/groups/my-org%2Fmy-subgroup", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/groups/my-org", httpCtx.Requests[1].URL.String())

		for _, req := range httpCtx.Requests[2:] {
			body, _ := io.ReadAll(req.Body)
			var reqBody map[string]any
			json.Unmarshal(body, &reqBody)
			variables := reqBody["variables"].(map[string]any)
			assert.Equal(t, "gid://gitlab/Group/456", variables["namespaceId"], "must use the root group's ID (456), not the subgroup's own ID (789)")
		}
	})

	t.Run("filters projects and recomputes totals when selected", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"projects": []string{"1"},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, ciMinutesNamespaceUsageResponseBody()),
					GitlabMockResponse(http.StatusOK, ciMinutesProjectUsageResponseBody()),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		payload := executionState.Payloads[0].(map[string]any)
		var result PipelineMinutesUsageResult
		payloadBytes, _ := json.Marshal(payload["data"])
		json.Unmarshal(payloadBytes, &result)

		require.Len(t, result.Projects, 1)
		assert.Equal(t, "hello-world", result.Projects[0].Project.Name)
		assert.Equal(t, 180, result.Minutes, "totals must reflect only the selected project")
		assert.Equal(t, 10800, result.SharedRunnersDuration)
	})

	t.Run("falls back to summed project usage when the namespace has no record for the month", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{"data": {"ciMinutesUsage": {"nodes": []}}}`),
					GitlabMockResponse(http.StatusOK, ciMinutesProjectUsageResponseBody()),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		payload := executionState.Payloads[0].(map[string]any)
		var result PipelineMinutesUsageResult
		payloadBytes, _ := json.Marshal(payload["data"])
		json.Unmarshal(payloadBytes, &result)

		require.Len(t, result.Projects, 2)
		assert.Equal(t, 245, result.Minutes, "totals must be summed from project usage, not silently left at zero")
		assert.Equal(t, 14700, result.SharedRunnersDuration)
		assert.NotEmpty(t, result.Month, "month must be derived from the query date, not left blank")
		assert.Equal(t, time.Now().Format("2006-01")+"-01", result.MonthIso8601)
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{"errors": [{"message": "not authorized to read usage"}]}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get pipeline minutes usage")
		assert.Contains(t, err.Error(), "not authorized to read usage")
	})

	t.Run("group resolution failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
					"groupId":     "my-org",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusNotFound, `{"message": "404 Group Not Found"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve group")

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		assert.Len(t, httpCtx.Requests, 1, "must not attempt the usage query if group resolution failed")
	})
}
