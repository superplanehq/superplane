package gitlab

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__AddIssueLabel__Setup(t *testing.T) {
	c := &AddIssueLabel{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"issueIid": "1",
				"labels":   []string{"bug"},
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
				"labels":  []string{"bug"},
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issue IID is required")
	})

	t.Run("at least one label is required", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
				"labels":   []string{},
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one label is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
				"labels":   []string{"bug", "urgent"},
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

func Test__AddIssueLabel__Execute(t *testing.T) {
	c := &AddIssueLabel{}

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

	t.Run("at least one label is required", func(t *testing.T) {
		err := c.Execute(core.ExecutionContext{
			Integration:    integration,
			ExecutionState: &contexts.ExecutionStateContext{},
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
				"labels":   []string{},
			},
		})

		require.ErrorContains(t, err, "at least one label is required")
	})

	t.Run("success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": 101,
					"iid": 1,
					"title": "Test Issue",
					"state": "opened",
					"labels": ["bug", "urgent", "needs-triage"]
				}`),
			},
		}

		err := c.Execute(core.ExecutionContext{
			Integration: integration,
			HTTP:        httpCtx,
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "1",
				"labels":   []string{"needs-triage"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		require.Len(t, executionState.Payloads, 1)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, "gitlab.labels", executionState.Type)

		payload := executionState.Payloads[0].(map[string]any)
		assert.Equal(t, []string{"bug", "urgent", "needs-triage"}, payload["data"])

		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodPut, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/issues/1", httpCtx.Requests[0].URL.String())

		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		assert.Contains(t, string(body), `"add_labels":"needs-triage"`)
	})

	t.Run("failure", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Issue Not Found"}`),
			},
		}

		err := c.Execute(core.ExecutionContext{
			Integration: integration,
			HTTP:        httpCtx,
			Configuration: map[string]any{
				"project":  "123",
				"issueIid": "999",
				"labels":   []string{"bug"},
			},
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "failed to add labels to issue")
	})
}
