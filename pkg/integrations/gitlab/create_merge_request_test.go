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

func Test__CreateMergeRequest__Setup(t *testing.T) {
	c := &CreateMergeRequest{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"sourceBranch": "feature/login-page",
				"targetBranch": "main",
				"title":        "feat: add login page",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing source branch", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":      "123",
				"targetBranch": "main",
				"title":        "feat: add login page",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source branch is required")
	})

	t.Run("missing target branch", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":      "123",
				"sourceBranch": "feature/login-page",
				"title":        "feat: add login page",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target branch is required")
	})

	t.Run("missing title", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":      "123",
				"sourceBranch": "feature/login-page",
				"targetBranch": "main",
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
				"project":      "123",
				"sourceBranch": "feature/login-page",
				"targetBranch": "main",
				"title":        "feat: add login page",
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

	t.Run("expression project is allowed", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":      "{{ $['On Push'].data.project.id }}",
				"sourceBranch": "{{ $['On Push'].data.ref }}",
				"targetBranch": "main",
				"title":        "feat: add login page",
			},
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__CreateMergeRequest__Execute(t *testing.T) {
	c := &CreateMergeRequest{}

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
				"project":      "123",
				"sourceBranch": "feature/login-page",
				"targetBranch": "main",
				"title":        "feat: add login page",
				"reviewers":    []string{"30"},
				"squash":       true,
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusCreated, `{
						"id": 1,
						"iid": 42,
						"project_id": 123,
						"title": "feat: add login page",
						"state": "opened",
						"created_at": "2026-02-13T08:46:00.000Z",
						"source_branch": "feature/login-page",
						"target_branch": "main",
						"reviewers": [{"id": 30, "username": "amorgan", "name": "Alex Morgan"}],
						"web_url": "https://gitlab.com/my-group/my-project/-/merge_requests/42"
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
		assert.Equal(t, "gitlab.mergeRequest", executionState.Type)

		var mergeRequest MergeRequest
		payloadBytes, _ := json.Marshal(payload["data"])
		json.Unmarshal(payloadBytes, &mergeRequest)

		assert.Equal(t, 42, mergeRequest.IID)
		assert.Equal(t, "opened", mergeRequest.State)
		assert.Equal(t, "feature/login-page", mergeRequest.SourceBranch)
		assert.Equal(t, "main", mergeRequest.TargetBranch)
		require.Len(t, mergeRequest.Reviewers, 1)
		assert.Equal(t, 30, mergeRequest.Reviewers[0].ID)
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":      "123",
				"sourceBranch": "feature/login-page",
				"targetBranch": "main",
				"title":        "feat: add login page",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusConflict, `{"message": ["Another open merge request already exists for this source branch"]}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create merge request")
	})
}
