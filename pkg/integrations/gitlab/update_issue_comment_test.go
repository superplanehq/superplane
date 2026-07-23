package gitlab

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateIssueComment__Setup(t *testing.T) {
	c := &UpdateIssueComment{}

	metadata := Metadata{Projects: []ProjectMetadata{{ID: 123, Name: "repo", URL: "http://repo"}}}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"issueIid": "1", "commentId": "302", "body": "Updated"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing issue IID", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "commentId": "302", "body": "Updated"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issue IID is required")
	})

	t.Run("missing comment ID", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "issueIid": "1", "body": "Updated"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "comment ID is required")
	})

	t.Run("missing body", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "issueIid": "1", "commentId": "302"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "body is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "issueIid": "1", "commentId": "302", "body": "Updated"},
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__UpdateIssueComment__Execute(t *testing.T) {
	c := &UpdateIssueComment{}

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
				"project":   "123",
				"issueIid":  "1",
				"commentId": "302",
				"body":      "Updated body",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{
						"id": 302,
						"body": "Updated body",
						"created_at": "2023-01-01T10:00:00.000Z",
						"updated_at": "2023-01-01T10:15:00.000Z",
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
		assert.Equal(t, "gitlab.updateIssueComment", executionState.Type)

		var note Note
		payloadBytes, _ := json.Marshal(payload["data"])
		json.Unmarshal(payloadBytes, &note)
		assert.Equal(t, 302, note.ID)
		assert.Equal(t, "Updated body", note.Body)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPut, httpCtx.Requests[0].Method)
		assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.Path, "/projects/123/issues/1/notes/302"))
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "Updated body", reqBody["body"])
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{"project": "123", "issueIid": "1", "commentId": "302", "body": "Updated body"},
			Integration:   integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusNotFound, `{"message": "404 Not found"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update issue comment")
	})
}
