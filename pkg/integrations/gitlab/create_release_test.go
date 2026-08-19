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

func Test__CreateRelease__Setup(t *testing.T) {
	c := &CreateRelease{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"tagName": "v1.0.0",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing tag name", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tag name is required")
	})

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project": "123",
				"tagName": "v1.0.0",
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

func Test__CreateRelease__Execute(t *testing.T) {
	c := &CreateRelease{}

	t.Run("success", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":     "123",
				"tagName":     "v1.0.0",
				"ref":         "main",
				"name":        "Release 1.0.0",
				"description": "First release",
				"milestones":  []string{"v1.0"},
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
						"tag_name": "v1.0.0",
						"name": "Release 1.0.0",
						"description": "First release"
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
		assert.Equal(t, "gitlab.release", executionState.Type)

		var release Release
		releasePayload := payload["data"]
		payloadBytes, _ := json.Marshal(releasePayload)
		json.Unmarshal(payloadBytes, &release)

		assert.Equal(t, "v1.0.0", release.TagName)
		assert.Equal(t, "Release 1.0.0", release.Name)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 1)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "v1.0.0", reqBody["tag_name"])
		assert.Equal(t, "main", reqBody["ref"])
		assert.Equal(t, []any{"v1.0"}, reqBody["milestones"])
	})

	t.Run("missing tag name", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"authType":    AuthTypePersonalAccessToken,
					"groupId":     "123",
					"accessToken": "pat",
					"baseUrl":     "https://gitlab.com",
				},
			},
			HTTP: &contexts.HTTPContext{},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tag name is required")

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		assert.Empty(t, httpCtx.Requests, "no request should be sent when the tag name is missing")
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project": "123",
				"tagName": "v1.0.0",
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
					GitlabMockResponse(http.StatusBadRequest, `{"message": "Tag name has already been taken"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create release")
	})
}
