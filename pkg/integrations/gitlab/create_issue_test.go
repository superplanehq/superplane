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

func Test__CreateIssue__Setup(t *testing.T) {
	c := &CreateIssue{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"title": "Issue Title",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing title", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
				"title":   "Issue Title",
			},
			Integration: &contexts.IntegrationContext{
				Metadata: Metadata{
					Repositories: []Repository{
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

func Test__CreateIssue__Execute(t *testing.T) {
	c := &CreateIssue{}

	t.Run("success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
				"title":   "Issue Title",
				"body":    "Issue Body",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":            AuthTypePersonalAccessToken,
					"groupId":             "123",
					"personalAccessToken": "pat",
					"baseUrl":             "https://gitlab.com",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusCreated, `{
						"id": 1,
						"title": "Issue Title",
						"description": "Issue Body",
						"web_url": "https://gitlab.com/issue/1"
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
		issuePayload := payload["data"]
		payloadBytes, _ := json.Marshal(issuePayload)
		json.Unmarshal(payloadBytes, &issue)

		assert.Equal(t, 1, issue.ID)
		assert.Equal(t, "Issue Title", issue.Title)
		assert.Equal(t, "Issue Body", issue.Description)
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
				"title":   "Issue Title",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":            AuthTypePersonalAccessToken,
					"groupId":             "123",
					"personalAccessToken": "pat",
					"baseUrl":             "https://gitlab.com",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusInternalServerError, `{"error": "internal server error"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create issue")
	})
}
