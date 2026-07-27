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

func Test__UpdateMergeRequest__Setup(t *testing.T) {
	c := &UpdateMergeRequest{}

	metadata := Metadata{Projects: []ProjectMetadata{{ID: 123, Name: "repo", URL: "http://repo"}}}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"mergeRequestIid": "42"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing merge request IID", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "merge request IID is required")
	})

	t.Run("invalid state", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "mergeRequestIid": "42", "state": "bogus"},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state")
	})

	t.Run("no fields enabled", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "mergeRequestIid": "42"},
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one field must be enabled")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "mergeRequestIid": "42", "title": "New title", "state": "close"},
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("labels toggled on but empty still counts as an update", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "mergeRequestIid": "42", "labels": []string{}},
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("title toggled on but empty is rejected", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "mergeRequestIid": "42", "title": ""},
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "title cannot be empty")
	})

	t.Run("target branch toggled on but empty is rejected", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{"project": "123", "mergeRequestIid": "42", "targetBranch": ""},
			Integration:   &contexts.IntegrationContext{Metadata: metadata},
			Metadata:      &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "target branch cannot be empty")
	})
}

func Test__UpdateMergeRequest__Execute(t *testing.T) {
	c := &UpdateMergeRequest{}

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
				"title":           "Updated Title",
				"description":     "Updated description",
				"targetBranch":    "develop",
				"state":           "reopen",
				"labels":          []string{"feature", "needs-review"},
				"assignees":       []string{"30"},
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{
						"id": 155,
						"iid": 42,
						"title": "Updated Title",
						"state": "opened",
						"target_branch": "develop"
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
		assert.Equal(t, "Updated Title", mergeRequest.Title)
		assert.Equal(t, "opened", mergeRequest.State)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPut, httpCtx.Requests[0].Method)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "Updated Title", reqBody["title"])
		assert.Equal(t, "Updated description", reqBody["description"])
		assert.Equal(t, "develop", reqBody["target_branch"])
		assert.Equal(t, "reopen", reqBody["state_event"])
		assert.Equal(t, "feature,needs-review", reqBody["labels"])
		assert.Equal(t, []any{float64(30)}, reqBody["assignee_ids"])
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{"project": "123", "mergeRequestIid": "42", "title": "Updated Title"},
			Integration:   integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusInternalServerError, `{"error": "internal server error"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update merge request")
	})

	t.Run("invalid assignee id", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{"project": "123", "mergeRequestIid": "42", "assignees": []string{"not-a-number"}},
			Integration:   integration,
			HTTP:          &contexts.HTTPContext{},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid assignee id")

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		assert.Empty(t, httpCtx.Requests, "no request should be sent when an assignee id is invalid")
	})

	t.Run("no fields enabled", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{"project": "123", "mergeRequestIid": "42"},
			Integration:   integration,
			HTTP:          &contexts.HTTPContext{},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one field must be enabled")
	})

	t.Run("clears labels and assignees when toggled on but empty", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"mergeRequestIid": "42",
				"description":     "",
				"labels":          []string{},
				"assignees":       []string{},
			},
			Integration: integration,
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{"id": 155, "iid": 42, "title": "MR"}`),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 1)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)

		assert.Equal(t, "", reqBody["description"])
		assert.Equal(t, "", reqBody["labels"])
		assert.Equal(t, []any{}, reqBody["assignee_ids"])

		// Fields that were never toggled on must be omitted entirely.
		assert.NotContains(t, reqBody, "title")
		assert.NotContains(t, reqBody, "state_event")
		assert.NotContains(t, reqBody, "target_branch")
	})
}
