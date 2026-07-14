package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetIssue__Setup(t *testing.T) {
	c := &GetIssue{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"issueIid": "7",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing issue IID", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issue IID is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "7",
			},
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{
					Projects: []ProjectMetadata{
						{ID: 123, Name: "repo", URL: "http://repo"},
					},
				},
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__GetIssue__Execute(t *testing.T) {
	c := &GetIssue{}

	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"authType":    AuthTypePersonalAccessToken,
			"groupId":     "123",
			"accessToken": "pat",
			"baseUrl":     "https://gitlab.com",
		},
	}

	t.Run("success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "7",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{
						"id": 41,
						"iid": 7,
						"project_id": 123,
						"title": "Login page rendering issue",
						"state": "opened",
						"labels": ["bug", "frontend"],
						"author": {"id": 22, "name": "Jamie Rivera", "username": "jrivera"},
						"web_url": "https://gitlab.com/my-group/my-project/-/issues/7"
					}`),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "gitlab.issue", executionState.Type)

		var issue Issue
		payloadBytes, _ := json.Marshal(payload["data"])
		json.Unmarshal(payloadBytes, &issue)

		assert.Equal(t, 7, issue.IID)
		assert.Equal(t, "Login page rendering issue", issue.Title)
		assert.Equal(t, "opened", issue.State)
		assert.Equal(t, []string{"bug", "frontend"}, issue.Labels)
		assert.Equal(t, "https://gitlab.com/my-group/my-project/-/issues/7", issue.WebURL)
	})

	t.Run("issue not found", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "999",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusNotFound, `{"message": "404 Not found"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get issue")
	})
}
