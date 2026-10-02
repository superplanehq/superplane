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

func Test__AddIssueAssignee__Setup(t *testing.T) {
	c := &AddIssueAssignee{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"issueIid":  "1",
				"assignees": []string{"30"},
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
				"project":   "123",
				"assignees": []string{"30"},
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issue IID is required")
	})

	t.Run("missing assignees", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one assignee is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":   "123",
				"issueIid":  "1",
				"assignees": []string{"30"},
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

func Test__AddIssueAssignee__Execute(t *testing.T) {
	c := &AddIssueAssignee{}

	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"authType":    AuthTypePersonalAccessToken,
			"groupId":     "123",
			"accessToken": "pat",
			"baseUrl":     "https://gitlab.com",
		},
	}

	t.Run("adds assignees to existing ones", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "123",
				"issueIid":  "1",
				"assignees": []string{"31"},
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{
						"id": 1,
						"iid": 1,
						"project_id": 123,
						"title": "Example Issue",
						"state": "opened",
						"assignees": [{"id": 30, "username": "amorgan", "name": "Alex Morgan"}],
						"web_url": "https://gitlab.com/my-group/my-project/-/issues/1"
					}`),
					GitlabMockResponse(http.StatusOK, `{
						"id": 1,
						"iid": 1,
						"project_id": 123,
						"title": "Example Issue",
						"state": "opened",
						"assignees": [
							{"id": 30, "username": "amorgan", "name": "Alex Morgan"},
							{"id": 31, "username": "schen", "name": "Sam Chen"}
						],
						"web_url": "https://gitlab.com/my-group/my-project/-/issues/1"
					}`),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		assert.Equal(t, "gitlab.updateIssue", executionState.Type)

		var issue Issue
		payloadBytes, _ := json.Marshal(payload["data"])
		json.Unmarshal(payloadBytes, &issue)

		require.Len(t, issue.Assignees, 2)
		assert.Equal(t, 30, issue.Assignees[0].ID)
		assert.Equal(t, 31, issue.Assignees[1].ID)
	})

	t.Run("fails when issue cannot be fetched", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "123",
				"issueIid":  "1",
				"assignees": []string{"31"},
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

	t.Run("fails when update fails", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "123",
				"issueIid":  "1",
				"assignees": []string{"31"},
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{"id": 1, "iid": 1, "assignees": []}`),
					GitlabMockResponse(http.StatusForbidden, `{"message": "403 Forbidden"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add issue assignees")
	})
}
