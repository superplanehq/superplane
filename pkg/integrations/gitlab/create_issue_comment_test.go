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

func Test__CreateIssueComment__Setup(t *testing.T) {
	c := &CreateIssueComment{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"issueIid": "1",
				"body":     "Comment body",
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
				"body":    "Comment body",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issue IID is required")
	})

	t.Run("missing body", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "body is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
				"body":     "Comment body",
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

func Test__CreateIssueComment__Execute(t *testing.T) {
	c := &CreateIssueComment{}

	t.Run("success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
				"body":     "Comment body",
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
					GitlabMockResponse(http.StatusCreated, `{
						"id": 302,
						"body": "Comment body",
						"created_at": "2023-01-01T10:00:00.000Z",
						"noteable_type": "Issue"
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
		assert.Equal(t, "gitlab.createIssueComment", executionState.Type)

		var note Note
		notePayload := payload["data"]
		payloadBytes, _ := json.Marshal(notePayload)
		json.Unmarshal(payloadBytes, &note)

		assert.Equal(t, 302, note.ID)
		assert.Equal(t, "Comment body", note.Body)
		assert.Equal(t, "Issue", note.NoteableType)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/issues/1/notes", httpCtx.Requests[0].URL.String())
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
				"body":     "Comment body",
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
		assert.Contains(t, err.Error(), "failed to create issue comment")
	})
}
