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

func Test__AcceptMergeRequest__Setup(t *testing.T) {
	c := &AcceptMergeRequest{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"mergeRequestIid": "42",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing merge request IID", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "merge request IID is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
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

func Test__AcceptMergeRequest__Execute(t *testing.T) {
	c := &AcceptMergeRequest{}

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
				"project":         "123",
				"mergeRequestIid": "42",
				"squash":          true,
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{
						"id": 1,
						"iid": 42,
						"project_id": 123,
						"title": "feat: add login page",
						"state": "merged",
						"merged_at": "2026-02-13T11:16:17.520Z",
						"source_branch": "feature/login-page",
						"target_branch": "main",
						"merge_commit_sha": "9999999999999999999999999999999999999999",
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
		assert.Equal(t, "merged", mergeRequest.State)
		assert.Equal(t, "main", mergeRequest.TargetBranch)
		require.NotNil(t, mergeRequest.MergeCommitSHA)
		assert.Equal(t, "9999999999999999999999999999999999999999", *mergeRequest.MergeCommitSHA)
	})

	t.Run("merge request cannot be merged", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusMethodNotAllowed, `{"message": "405 Method Not Allowed"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "merge request cannot be merged")
	})

	t.Run("user not allowed to merge", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusUnauthorized, `{"message": "401 Unauthorized"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not have permission to accept this merge request")
	})

	t.Run("branch cannot be merged", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusUnprocessableEntity, `{"message": "Branch cannot be merged"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch cannot be merged")
	})

	t.Run("SHA mismatch", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
				"sha":             "0000000000000000000000000000000000000000",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusConflict, `{"message": "SHA does not match HEAD of source branch"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SHA does not match HEAD of source branch")
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusInternalServerError, `{"error": "internal server error"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to accept merge request")
	})
}
