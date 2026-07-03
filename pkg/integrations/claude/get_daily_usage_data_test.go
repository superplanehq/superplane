package claude

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetDailyUsageData__Execute(t *testing.T) {
	c := &GetDailyUsageData{}

	messagesResponse := `{
		"data": [
			{
				"starting_at": "2024-03-18T00:00:00Z",
				"ending_at": "2024-03-19T00:00:00Z",
				"results": [
					{
						"model": "claude-sonnet-5",
						"uncached_input_tokens": 1000,
						"output_tokens": 200,
						"cache_read_input_tokens": 50,
						"cache_creation": {"ephemeral_1h_input_tokens": 10, "ephemeral_5m_input_tokens": 5},
						"server_tool_use": {"web_search_requests": 2}
					}
				]
			}
		],
		"has_more": false,
		"next_page": null
	}`

	claudeCodeResponse := `{
		"data": [
			{
				"actor": {"type": "user_actor", "email_address": "dev@company.com"},
				"date": "2024-03-18",
				"core_metrics": {
					"commits_by_claude_code": 3,
					"lines_of_code": {"added": 100, "removed": 40},
					"num_sessions": 5,
					"pull_requests_by_claude_code": 1
				},
				"model_breakdown": [
					{
						"model": "claude-sonnet-5",
						"estimated_cost": {"amount": 186, "currency": "USD"},
						"tokens": {"cache_creation": 10, "cache_read": 20, "input": 300, "output": 80}
					}
				],
				"tool_actions": {
					"edit_tool": {"accepted": 4, "rejected": 1}
				}
			}
		],
		"has_more": false,
		"next_page": null
	}`

	t.Run("success with custom single-day range", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(messagesResponse))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(claudeCodeResponse))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiKey":   "sk-123",
				"adminKey": "sk-ant-admin-123",
			},
		}

		executionStateCtx := &contexts.ExecutionStateContext{}

		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"startDate": "2024-03-18",
				"endDate":   "2024-03-18",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionStateCtx,
			Logger:         logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/organizations/usage_report/messages")
		assert.Equal(t, "sk-ant-admin-123", httpContext.Requests[0].Header.Get("x-api-key"))
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/organizations/usage_report/claude_code")

		assert.Equal(t, core.DefaultOutputChannel.Name, executionStateCtx.Channel)
		assert.Equal(t, GetDailyUsageDataPayloadType, executionStateCtx.Type)

		require.Len(t, executionStateCtx.Payloads, 1)
		wrapped := executionStateCtx.Payloads[0].(map[string]any)
		output := wrapped["data"].(GetDailyUsageDataOutput)

		assert.Equal(t, "2024-03-18", output.Period.StartDate)
		assert.Equal(t, "2024-03-18", output.Period.EndDate)

		assert.Equal(t, int64(1000), output.Messages.InputTokens)
		assert.Equal(t, int64(200), output.Messages.OutputTokens)
		assert.Equal(t, int64(50), output.Messages.CacheReadTokens)
		assert.Equal(t, int64(15), output.Messages.CacheCreationTokens)
		assert.Equal(t, int64(2), output.Messages.WebSearchRequests)
		require.Len(t, output.Messages.ByModel, 1)
		assert.Equal(t, "claude-sonnet-5", output.Messages.ByModel[0].Model)

		assert.Equal(t, int64(5), output.ClaudeCode.Sessions)
		assert.Equal(t, int64(100), output.ClaudeCode.LinesAdded)
		assert.Equal(t, int64(40), output.ClaudeCode.LinesRemoved)
		assert.Equal(t, int64(3), output.ClaudeCode.Commits)
		assert.Equal(t, int64(1), output.ClaudeCode.PullRequests)
		assert.Equal(t, int64(4), output.ClaudeCode.ToolActionsAccepted)
		assert.Equal(t, int64(1), output.ClaudeCode.ToolActionsRejected)
		assert.InDelta(t, 1.86, output.ClaudeCode.EstimatedCostUsd, 0.001)
		require.Len(t, output.ClaudeCode.ByActor, 1)
		assert.Equal(t, "dev@company.com", output.ClaudeCode.ByActor[0].Actor)

		require.Len(t, output.Daily, 1)
		assert.Equal(t, "2024-03-18", output.Daily[0].Date)
		assert.Equal(t, int64(1000), output.Daily[0].MessagesInputTokens)
		assert.Equal(t, int64(5), output.Daily[0].CodeSessions)
	})

	t.Run("invalid start date format -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"adminKey": "test-admin-key"},
		}
		execCtx := core.ExecutionContext{
			ID:            uuid.New(),
			Configuration: map[string]any{"startDate": "invalid-date"},
			HTTP:          httpContext,
			Integration:   integrationCtx,
			Logger:        logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid start date format")
	})

	t.Run("invalid end date format -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"adminKey": "test-admin-key"},
		}
		execCtx := core.ExecutionContext{
			ID:            uuid.New(),
			Configuration: map[string]any{"endDate": "invalid-date"},
			HTTP:          httpContext,
			Integration:   integrationCtx,
			Logger:        logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid end date format")
	})

	t.Run("start date after end date -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"adminKey": "test-admin-key"},
		}
		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"startDate": "2024-03-25",
				"endDate":   "2024-03-20",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "start date must be before end date")
	})

	t.Run("date range too large -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"adminKey": "test-admin-key"},
		}
		execCtx := core.ExecutionContext{
			ID: uuid.New(),
			Configuration: map[string]any{
				"startDate": "2024-01-01",
				"endDate":   "2024-03-01",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Logger:      logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "date range cannot exceed")
	})

	t.Run("missing admin key -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "sk-123"},
		}
		execCtx := core.ExecutionContext{
			ID:            uuid.New(),
			Configuration: map[string]any{},
			HTTP:          httpContext,
			Integration:   integrationCtx,
			Logger:        logrus.NewEntry(logrus.New()),
		}

		err := c.Execute(execCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "admin API key is not configured")
	})
}
