package gitlab

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GitLab__ListResources(t *testing.T) {
	g := &GitLab{}

	t.Run("returns empty list for unknown resource type", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{}
		resources, err := g.ListResources("unknown", core.ListResourcesContext{
			Integration: ctx,
		})
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("returns list of members", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseUrl":     "https://gitlab.com",
					"groupId":     "123",
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "token",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `[
						{"id": 101, "name": "User One", "username": "user1"},
						{"id": 102, "name": "User Two", "username": "user2"}
					]`),
				},
			},
		}

		resources, err := g.ListResources("member", ctx)

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "101", resources[0].ID)
		assert.Equal(t, "User One (@user1)", resources[0].Name)
		assert.Equal(t, "member", resources[0].Type)
		assert.Equal(t, "102", resources[1].ID)
	})

	t.Run("returns list of projects from metadata", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Projects: []ProjectMetadata{
					{ID: 1, Name: "project1", URL: "http://project1"},
					{ID: 2, Name: "project2", URL: "http://project2"},
				},
			},
		}

		resources, err := g.ListResources("project", core.ListResourcesContext{
			Integration: ctx,
		})

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "1", resources[0].ID)
		assert.Equal(t, "project1", resources[0].Name)
		assert.Equal(t, "project", resources[0].Type)
		assert.Equal(t, "2", resources[1].ID)
		assert.Equal(t, "project2", resources[1].Name)
	})

	t.Run("handles invalid metadata gracefully", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Metadata: "invalid-string-metadata",
		}

		_, err := g.ListResources("project", core.ListResourcesContext{
			Integration: ctx,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode metadata")
	})

	t.Run("returns list of milestones for project", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseUrl":     "https://gitlab.com",
					"groupId":     "123",
					"authType":    AuthTypePersonalAccessToken,
					"accessToken": "token",
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					GitlabMockResponse(http.StatusOK, `[
						{"id": 1, "iid": 1, "title": "v1.0", "state": "active"},
						{"id": 2, "iid": 2, "title": "v2.0", "state": "active"}
					]`),
				},
			},
			Parameters: map[string]string{
				"project": "456",
			},
		}

		resources, err := g.ListResources("milestone", ctx)

		require.NoError(t, err)
		assert.Len(t, resources, 2)
		assert.Equal(t, "1", resources[0].ID)
		assert.Equal(t, "v1.0", resources[0].Name)
		assert.Equal(t, "milestone", resources[0].Type)
		assert.Equal(t, "2", resources[1].ID)
		assert.Equal(t, "v2.0", resources[1].Name)
	})

	t.Run("returns empty list for milestone without project", func(t *testing.T) {
		ctx := core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{},
			Parameters:  map[string]string{},
		}

		resources, err := g.ListResources("milestone", ctx)

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

}
