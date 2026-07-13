package gitlab

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateIssue__Setup(t *testing.T) {
	c := &UpdateIssue{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"issueIid": "1",
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

	t.Run("invalid state", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
				"state":    "bogus",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
				"title":    "New title",
				"state":    "close",
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

func Test__UpdateIssue__Execute(t *testing.T) {
	c := &UpdateIssue{}

	t.Run("success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "123",
				"issueIid":  "1",
				"title":     "Updated Title",
				"state":     "close",
				"labels":    []string{"bug"},
				"assignees": []string{"99"},
				"milestone": "12",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"groupId":     "123",
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{
						"id": 101,
						"iid": 1,
						"title": "Updated Title",
						"state": "closed"
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
		assert.Equal(t, "gitlab.updateIssue", executionState.Type)

		var issue Issue
		issuePayload := payload["data"]
		payloadBytes, _ := json.Marshal(issuePayload)
		json.Unmarshal(payloadBytes, &issue)

		assert.Equal(t, "Updated Title", issue.Title)
		assert.Equal(t, "closed", issue.State)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 1)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "close", reqBody["state_event"])
		assert.Equal(t, "bug", reqBody["labels"])
		assert.Equal(t, float64(12), reqBody["milestone_id"])
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
				"title":    "Updated Title",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"groupId":     "123",
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
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
		assert.Contains(t, err.Error(), "failed to update issue")
	})
}
