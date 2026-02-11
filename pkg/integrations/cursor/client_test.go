package cursor

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__VerifyLaunchAgent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"agents":[]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"launchAgentKey": "test-key",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		err = client.VerifyLaunchAgent()
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "Bearer test-key", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("unauthorized", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":"invalid key"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"launchAgentKey": "invalid-key",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		err = client.VerifyLaunchAgent()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired")
	})
}

func ptrBool(b bool) *bool { return &b }

func Test__Client__LaunchAgent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "agent-123",
						"status": "CREATING",
						"source": {"repository": "https://github.com/org/repo", "ref": "main"},
						"target": {"branchName": "cursor/agent-abc123"}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"launchAgentKey": "test-key",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		req := launchAgentRequest{
			Prompt: launchAgentPrompt{Text: "Fix the bug"},
			Source: launchAgentSource{
				Repository: "https://github.com/org/repo",
				Ref:        "main",
			},
			Target: launchAgentTarget{
				AutoCreatePr: ptrBool(true),
				BranchName:   "cursor/agent-abc123",
			},
		}

		response, err := client.LaunchAgent(req)
		require.NoError(t, err)

		assert.Equal(t, "agent-123", response.ID)
		assert.Equal(t, "CREATING", response.Status)
		assert.Equal(t, "cursor/agent-abc123", response.Target.BranchName)
	})

	t.Run("no cloud agent key", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		_, err = client.LaunchAgent(launchAgentRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cloud Agent API key is not configured")
	})
}

func Test__Client__GetAgentStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id": "agent-123",
						"status": "FINISHED",
						"summary": "Fixed the bug",
						"target": {"prUrl": "https://github.com/org/repo/pull/42"}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"launchAgentKey": "test-key",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		response, err := client.GetAgentStatus("agent-123")
		require.NoError(t, err)

		assert.Equal(t, "agent-123", response.ID)
		assert.Equal(t, "FINISHED", response.Status)
		assert.Equal(t, "Fixed the bug", response.Summary)
		assert.Equal(t, "https://github.com/org/repo/pull/42", response.Target.PrURL)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cursor.com/v0/agents/agent-123", httpContext.Requests[0].URL.String())
	})
}

func Test__Client__GetDailyUsage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": [
							{
								"date": 1710720000000,
								"isActive": true,
								"totalLinesAdded": 1543,
								"email": "dev@company.com"
							}
						],
						"period": {
							"startDate": 1710720000000,
							"endDate": 1710892800000
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"adminKey": "test-admin-key",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		req := UsageRequest{
			StartDate: 1710720000000,
			EndDate:   1710892800000,
		}

		response, err := client.GetDailyUsage(req)
		require.NoError(t, err)

		data := (*response)["data"].([]any)
		assert.Len(t, data, 1)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.cursor.com/teams/daily-usage-data", httpContext.Requests[0].URL.String())
		assert.Equal(t, "Bearer test-admin-key", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("no admin key", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		_, err = client.GetDailyUsage(UsageRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Admin API key is not configured")
	})
}

func Test__Client__ListModels(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"models":["claude-3.5-sonnet","gpt-4o","o1-mini"]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"launchAgentKey": "test-key",
			},
		}

		client, err := NewClient(httpContext, integrationCtx)
		require.NoError(t, err)

		models, err := client.ListModels()
		require.NoError(t, err)

		assert.Len(t, models, 3)
		assert.Contains(t, models, "claude-3.5-sonnet")
		assert.Contains(t, models, "gpt-4o")
		assert.Contains(t, models, "o1-mini")
	})
}
