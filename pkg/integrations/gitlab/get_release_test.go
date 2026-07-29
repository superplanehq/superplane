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

func Test__GetRelease__Setup(t *testing.T) {
	c := &GetRelease{}

	t.Run("missing project", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
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

func Test__GetRelease__Execute(t *testing.T) {
	c := &GetRelease{}

	t.Run("success - specific tag", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
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
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `{"tag_name": "v1.0.0", "name": "Release 1.0.0"}`),
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
		payloadBytes, _ := json.Marshal(payload["data"])
		json.Unmarshal(payloadBytes, &release)
		assert.Equal(t, "v1.0.0", release.TagName)

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/releases/v1.0.0", httpCtx.Requests[0].URL.String())
	})

	t.Run("success - latest release strategy resolves tag first", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategyLatest,
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
					GitlabMockResponse(http.StatusOK, `{"tag_name": "v2.0.0"}`),
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

	t.Run("missing tag name", func(t *testing.T) {
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
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
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("failure", func(t *testing.T) {
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
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusNotFound, `{"message": "404 Not Found"}`),
				},
			},
		}

		err := c.Execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get release")
	})
}
