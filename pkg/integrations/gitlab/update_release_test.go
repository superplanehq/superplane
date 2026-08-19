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

func Test__UpdateRelease__Setup(t *testing.T) {
	c := &UpdateRelease{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
				"name":            "New name",
			},
			Metadata: &contexts.MetadataContext{},
		}
		err := c.Setup(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("missing tag name for specific strategy", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"name":            "New name",
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
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tag name is required")
	})

	t.Run("tag name not required for latest strategy", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategyLatest,
				"name":            "New name",
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

	t.Run("no fields enabled", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
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
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one field must be enabled")
	})

	t.Run("released at toggled on but empty is rejected", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
				"releasedAt":      "",
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
		require.Error(t, err)
		assert.Contains(t, err.Error(), "released at cannot be empty")
	})

	t.Run("milestones toggled on but empty still counts as an update", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
				"milestones":      []string{},
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

func Test__UpdateRelease__Execute(t *testing.T) {
	c := &UpdateRelease{}

	t.Run("success - specific tag", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
				"name":            "Updated name",
				"milestones":      []string{"v1.0"},
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
					GitlabMockResponse(http.StatusOK, `{"tag_name": "v1.0.0", "name": "Updated name"}`),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		assert.Equal(t, "gitlab.release", executionState.Type)

		var release Release
		releasePayload := payload["data"]
		payloadBytes, _ := json.Marshal(releasePayload)
		json.Unmarshal(payloadBytes, &release)
		assert.Equal(t, "Updated name", release.Name)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/releases/v1.0.0", httpCtx.Requests[0].URL.String())

		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, "Updated name", reqBody["name"])
		assert.Equal(t, []any{"v1.0"}, reqBody["milestones"])
		assert.NotContains(t, reqBody, "description")
	})

	t.Run("success - latest release strategy resolves tag first", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategyLatest,
				"description":     "Updated notes",
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
					GitlabMockResponse(http.StatusOK, `[{"tag_name": "v2.0.0"}]`),
					GitlabMockResponse(http.StatusOK, `{"tag_name": "v2.0.0", "description": "Updated notes"}`),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 2)
		assert.Contains(t, httpCtx.Requests[0].URL.String(), "/releases?order_by=released_at")
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/releases/v2.0.0", httpCtx.Requests[1].URL.String())
	})

	t.Run("no fields enabled", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
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
		assert.Contains(t, err.Error(), "at least one field must be enabled")

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("clears milestones when toggled on but empty", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
				"milestones":      []string{},
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
					GitlabMockResponse(http.StatusOK, `{"tag_name": "v1.0.0"}`),
				},
			},
			ExecutionState: executionState,
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		body, _ := io.ReadAll(httpCtx.Requests[0].Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		assert.Equal(t, []any{}, reqBody["milestones"])
	})

	t.Run("failure", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
				"name":            "New name",
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
					GitlabMockResponse(http.StatusNotFound, `{"message": "404 Not Found"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update release")
	})
}
