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

func draftMergeRequestResponse() *http.Response {
	return GitlabMockResponse(http.StatusOK, `{
		"id": 1,
		"iid": 42,
		"project_id": 123,
		"title": "feat: add login page",
		"state": "opened",
		"draft": true,
		"web_url": "https://gitlab.com/my-group/my-project/-/merge_requests/42"
	}`)
}

func readyMergeRequestResponse() *http.Response {
	return GitlabMockResponse(http.StatusOK, `{
		"id": 1,
		"iid": 42,
		"project_id": 123,
		"title": "feat: add login page",
		"state": "opened",
		"draft": false,
		"web_url": "https://gitlab.com/my-group/my-project/-/merge_requests/42"
	}`)
}

func Test__MarkMergeRequestReadyForReview__Setup(t *testing.T) {
	c := &MarkMergeRequestReadyForReview{}

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

	t.Run("expression project and IID are allowed", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":         "{{ $['On Merge Request'].data.project.id }}",
				"mergeRequestIid": "{{ $['On Merge Request'].data.object_attributes.iid }}",
			},
			Integration: &contexts.IntegrationContext{},
			Metadata:    &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__MarkMergeRequestReadyForReview__Execute(t *testing.T) {
	c := &MarkMergeRequestReadyForReview{}

	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"authType":    AuthTypePersonalAccessToken,
			"accessToken": "pat",
			"baseUrl":     "https://gitlab.com",
		},
	}

	t.Run("fails when configuration decode fails", func(t *testing.T) {
		err := c.Execute(core.ExecutionContext{
			Integration:    integration,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration:  "not a map",
		})

		require.ErrorContains(t, err, "failed to decode configuration")
	})

	t.Run("marks a draft merge request ready for review", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				draftMergeRequestResponse(),
				// GitLab returns 202 (not 201) for a note whose body is only
				// a quick action, with a commands summary instead of a note.
				GitlabMockResponse(http.StatusAccepted, `{"commands_changes":{"wip_event":"ready"},"summary":["Marked this merge request as ready."]}`),
				readyMergeRequestResponse(),
			},
		}

		err := c.Execute(core.ExecutionContext{
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "gitlab.mergeRequest", executionState.Type)
		require.Len(t, executionState.Payloads, 1)
		require.Len(t, httpCtx.Requests, 3)

		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/merge_requests/42", httpCtx.Requests[0].URL.String())

		noteRequest := httpCtx.Requests[1]
		assert.Equal(t, http.MethodPost, noteRequest.Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/merge_requests/42/notes", noteRequest.URL.String())
		body, _ := io.ReadAll(noteRequest.Body)
		assert.Contains(t, string(body), `"body":"/ready"`)

		assert.Equal(t, http.MethodGet, httpCtx.Requests[2].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/merge_requests/42", httpCtx.Requests[2].URL.String())

		payload := executionState.Payloads[0].(map[string]any)
		payloadBytes, _ := json.Marshal(payload["data"])
		var mergeRequest MergeRequest
		json.Unmarshal(payloadBytes, &mergeRequest)
		assert.Equal(t, 42, mergeRequest.IID)
		assert.False(t, mergeRequest.Draft)
	})

	t.Run("merge request that is already ready is emitted without the note", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				readyMergeRequestResponse(),
			},
		}

		err := c.Execute(core.ExecutionContext{
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: executionState,
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
			},
		})

		require.NoError(t, err)
		require.True(t, executionState.Passed)
		assert.Equal(t, "gitlab.mergeRequest", executionState.Type)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)

		payload := executionState.Payloads[0].(map[string]any)
		payloadBytes, _ := json.Marshal(payload["data"])
		var mergeRequest MergeRequest
		json.Unmarshal(payloadBytes, &mergeRequest)
		assert.False(t, mergeRequest.Draft)
	})

	t.Run("fails when the merge request cannot be fetched", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Merge Request Not Found"}`),
			},
		}

		err := c.Execute(core.ExecutionContext{
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
			},
		})

		require.ErrorContains(t, err, "failed to get merge request")
	})

	t.Run("fails when the ready quick action cannot be applied", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				draftMergeRequestResponse(),
				GitlabMockResponse(http.StatusForbidden, `{"message": "403 Forbidden"}`),
			},
		}

		err := c.Execute(core.ExecutionContext{
			Integration:    integration,
			HTTP:           httpCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
			},
		})

		require.ErrorContains(t, err, "failed to mark merge request ready for review")
	})
}
