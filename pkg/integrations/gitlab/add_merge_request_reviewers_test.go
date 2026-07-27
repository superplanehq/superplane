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

func Test__AddMergeRequestReviewers__Setup(t *testing.T) {
	c := &AddMergeRequestReviewers{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"mergeRequestIid": "42",
				"reviewers":       []string{"30"},
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
				"project":   "123",
				"reviewers": []string{"30"},
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "merge request IID is required")
	})

	t.Run("missing reviewers", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one reviewer is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
				"reviewers":       []string{"30"},
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

func Test__AddMergeRequestReviewers__Execute(t *testing.T) {
	c := &AddMergeRequestReviewers{}

	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"authType":    AuthTypePersonalAccessToken,
			"groupId":     "123",
			"accessToken": "pat",
			"baseUrl":     "https://gitlab.com",
		},
	}

	t.Run("adds reviewers to existing ones", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
				"reviewers":       []string{"31"},
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{
						"id": 1,
						"iid": 42,
						"project_id": 123,
						"title": "feat: add login page",
						"state": "opened",
						"reviewers": [{"id": 30, "username": "amorgan", "name": "Alex Morgan"}],
						"web_url": "https://gitlab.com/my-group/my-project/-/merge_requests/42"
					}`),
					GitlabMockResponse(http.StatusOK, `{
						"id": 1,
						"iid": 42,
						"project_id": 123,
						"title": "feat: add login page",
						"state": "opened",
						"reviewers": [
							{"id": 30, "username": "amorgan", "name": "Alex Morgan"},
							{"id": 31, "username": "schen", "name": "Sam Chen"}
						],
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
		assert.Equal(t, "gitlab.mergeRequest", executionState.Type)

		var mergeRequest MergeRequest
		payloadBytes, _ := json.Marshal(payload["data"])
		json.Unmarshal(payloadBytes, &mergeRequest)

		require.Len(t, mergeRequest.Reviewers, 2)
		assert.Equal(t, 30, mergeRequest.Reviewers[0].ID)
		assert.Equal(t, 31, mergeRequest.Reviewers[1].ID)
	})

	t.Run("fails when merge request cannot be fetched", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
				"reviewers":       []string{"31"},
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
		assert.Contains(t, err.Error(), "failed to get merge request")
	})

	t.Run("fails when update fails", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
				"reviewers":       []string{"31"},
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{"id": 1, "iid": 42, "reviewers": []}`),
					GitlabMockResponse(http.StatusForbidden, `{"message": "403 Forbidden"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add merge request reviewers")
	})
}
