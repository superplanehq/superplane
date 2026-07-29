package gitlab

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteRelease__Setup(t *testing.T) {
	c := &DeleteRelease{}

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
}

func Test__DeleteRelease__Execute(t *testing.T) {
	c := &DeleteRelease{}

	t.Run("success - deletes release and tag", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
				"deleteTag":       true,
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
					GitlabMockResponse(http.StatusNoContent, ``),
				},
			},
			ExecutionState: executionState,
			Logger:         log.NewEntry(log.New()),
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		assert.Equal(t, "gitlab.release", executionState.Type)

		data := payload["data"].(map[string]any)
		assert.Equal(t, "v1.0.0", data["tag_name"])
		assert.Equal(t, true, data["tag_deleted"])

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, http.MethodDelete, httpCtx.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/releases/v1.0.0", httpCtx.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/projects/123/repository/tags/v1.0.0", httpCtx.Requests[1].URL.String())
	})

	t.Run("success - does not delete tag when disabled", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
				"deleteTag":       false,
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
			Logger:         log.NewEntry(log.New()),
		}

		err := c.Execute(ctx)
		require.NoError(t, err)

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, false, data["tag_deleted"], "tag_deleted must be false when tag deletion wasn't requested")

		httpCtx := ctx.HTTP.(*contexts.HTTPContext)
		require.Len(t, httpCtx.Requests, 1)
	})

	t.Run("release deleted even if tag deletion fails", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{}
		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"project":         "123",
				"releaseStrategy": ReleaseStrategySpecific,
				"tagName":         "v1.0.0",
				"deleteTag":       true,
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
					GitlabMockResponse(http.StatusNotFound, `{"message": "404 Tag Not Found"}`),
				},
			},
			ExecutionState: executionState,
			Logger:         log.NewEntry(log.New()),
		}

		err := c.Execute(ctx)
		require.NoError(t, err, "the release deletion already succeeded, so a failed tag deletion should not fail the operation")

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, false, data["tag_deleted"], "a failed tag deletion must be visible in the output, not just logged")
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
		assert.Contains(t, err.Error(), "failed to delete release")
	})
}
