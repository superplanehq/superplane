package gitlab

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__NewClient(t *testing.T) {
	mockClient := &contexts.HTTPContext{}

	t.Run("valid configuration - personal access token", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":            AuthTypePersonalAccessToken,
				"baseUrl":             "https://gitlab.example.com",
				"personalAccessToken": "pat-123",
				"groupId":             "group-123",
			},
		}

		client, err := NewClient(mockClient, ctx)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "https://gitlab.example.com", client.baseURL)
		assert.Equal(t, "pat-123", client.token)
		assert.Equal(t, AuthTypePersonalAccessToken, client.authType)
		assert.Equal(t, "group-123", client.groupID)
		assert.Equal(t, mockClient, client.httpClient)
	})

	t.Run("valid configuration - oauth", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypeAppOAuth,
				"groupId":  "group-456",
			},
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("oauth-token-123")},
			},
		}

		client, err := NewClient(mockClient, ctx)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, "https://gitlab.com", client.baseURL)
		assert.Equal(t, "oauth-token-123", client.token)
		assert.Equal(t, AuthTypeAppOAuth, client.authType)
		assert.Equal(t, "group-456", client.groupID)
	})

	t.Run("missing authType", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{},
		}
		_, err := NewClient(mockClient, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get authType")
	})

	t.Run("missing groupId", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypePersonalAccessToken,
			},
		}
		_, err := NewClient(mockClient, ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "groupId is required")
	})

	t.Run("missing personal access token", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType": AuthTypePersonalAccessToken,
				"groupId":  "123",
			},
		}
		_, err := NewClient(mockClient, ctx)
		require.Error(t, err)
	})
}

func Test__Client__FetchIntegrationData(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 1, "username": "user1"}`),
				GitlabMockResponse(http.StatusOK, `[{"id": 1, "path_with_namespace": "group/project1", "web_url": "https://gitlab.com/group/project1"}]`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		user, projects, err := client.FetchIntegrationData()
		require.NoError(t, err)

		require.Len(t, mockClient.Requests, 2)
		assert.Equal(t, "https://gitlab.com/api/v4/user", mockClient.Requests[0].URL.String())
		assert.Equal(t, "token", mockClient.Requests[0].Header.Get("PRIVATE-TOKEN"))

		assert.Equal(t, "https://gitlab.com/api/v4/groups/123/projects?include_subgroups=true&per_page=100&page=1", mockClient.Requests[1].URL.String())
		assert.Equal(t, "token", mockClient.Requests[1].Header.Get("PRIVATE-TOKEN"))

		require.NotNil(t, user)
		assert.Equal(t, 1, user.ID)
		assert.Equal(t, "user1", user.Username)

		require.Len(t, projects, 1)
		assert.Equal(t, 1, projects[0].ID)
		assert.Equal(t, "group/project1", projects[0].PathWithNamespace)
	})

	t.Run("pagination", func(t *testing.T) {
		resp1 := GitlabMockResponse(http.StatusOK, `[{"id": 1}]`)
		resp1.Header.Set("X-Next-Page", "2")

		resp2 := GitlabMockResponse(http.StatusOK, `[{"id": 2}]`)

		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 1, "username": "user1"}`),
				resp1,
				resp2,
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, projects, err := client.FetchIntegrationData()
		require.NoError(t, err)

		require.Len(t, mockClient.Requests, 3)
		assert.Equal(t, "https://gitlab.com/api/v4/user", mockClient.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123/projects?include_subgroups=true&per_page=100&page=1", mockClient.Requests[1].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123/projects?include_subgroups=true&per_page=100&page=2", mockClient.Requests[2].URL.String())

		require.Len(t, projects, 2)
		assert.Equal(t, 1, projects[0].ID)
		assert.Equal(t, 2, projects[1].ID)
	})

	t.Run("forbidden", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 1}`),
				GitlabMockResponse(http.StatusForbidden, `{"error": "fraud"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, _, err := client.FetchIntegrationData()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 403")
	})

	t.Run("oauth headers", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 1}`),
				GitlabMockResponse(http.StatusOK, `[]`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "oauth-token",
			authType:   AuthTypeAppOAuth,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, _, err := client.FetchIntegrationData()
		require.NoError(t, err)

		require.Len(t, mockClient.Requests, 2)
		assert.Equal(t, "Bearer oauth-token", mockClient.Requests[0].Header.Get("Authorization"))
	})
}

func Test__Client__CreateIssue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{
					"id": 101, 
					"iid": 1, 
					"title": "Test Issue", 
					"web_url": "https://gitlab.com/group/project/issues/1",
					"due_date": "2023-10-27",
					"milestone": {"id": 12, "title": "v1.0"},
					"closed_at": "2023-10-28T10:00:00Z",
					"closed_by": {"id": 5, "username": "closer"}
				}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		req := &IssueRequest{Title: "Test Issue"}
		result, err := client.CreateIssue(context.Background(), "1", req)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 101, result.ID)
		assert.Equal(t, "Test Issue", result.Title)

		require.NotNil(t, result.DueDate)
		assert.Equal(t, "2023-10-27", *result.DueDate)

		require.NotNil(t, result.Milestone)
		assert.Equal(t, 12, result.Milestone.ID)
		assert.Equal(t, "v1.0", result.Milestone.Title)

		require.NotNil(t, result.ClosedAt)
		assert.Equal(t, "2023-10-28T10:00:00Z", *result.ClosedAt)

		require.NotNil(t, result.ClosedBy)
		assert.Equal(t, 5, result.ClosedBy.ID)
		assert.Equal(t, "closer", result.ClosedBy.Username)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPost, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/issues", mockClient.Requests[0].URL.String())
	})
}

func Test__Client__ListGroupMembers(t *testing.T) {
	t.Run("pagination", func(t *testing.T) {
		resp1 := GitlabMockResponse(http.StatusOK, `[{"id": 1, "username": "user1"}]`)
		resp1.Header.Set("X-Next-Page", "2")

		resp2 := GitlabMockResponse(http.StatusOK, `[{"id": 2, "username": "user2"}]`)

		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				resp1,
				resp2,
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		members, err := client.ListGroupMembers("123")
		require.NoError(t, err)

		require.Len(t, mockClient.Requests, 2)
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123/members?per_page=100&page=1", mockClient.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/groups/123/members?per_page=100&page=2", mockClient.Requests[1].URL.String())

		require.Len(t, members, 2)
		assert.Equal(t, "user1", members[0].Username)
		assert.Equal(t, "user2", members[1].Username)
	})
}

func Test__Client__ListMilestones(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[
					{"id": 1, "iid": 1, "title": "v1.0", "state": "active"},
					{"id": 2, "iid": 2, "title": "v2.0", "state": "active"}
				]`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		milestones, err := client.ListMilestones("456")
		require.NoError(t, err)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/milestones?per_page=100&page=1&state=active", mockClient.Requests[0].URL.String())
		assert.Equal(t, "token", mockClient.Requests[0].Header.Get("PRIVATE-TOKEN"))

		require.Len(t, milestones, 2)
		assert.Equal(t, 1, milestones[0].ID)
		assert.Equal(t, "v1.0", milestones[0].Title)
		assert.Equal(t, 2, milestones[1].ID)
		assert.Equal(t, "v2.0", milestones[1].Title)
	})

	t.Run("pagination", func(t *testing.T) {
		resp1 := GitlabMockResponse(http.StatusOK, `[{"id": 1, "iid": 1, "title": "v1.0", "state": "active"}]`)
		resp1.Header.Set("X-Next-Page", "2")

		resp2 := GitlabMockResponse(http.StatusOK, `[{"id": 2, "iid": 2, "title": "v2.0", "state": "active"}]`)

		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				resp1,
				resp2,
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		milestones, err := client.ListMilestones("456")
		require.NoError(t, err)

		require.Len(t, mockClient.Requests, 2)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/milestones?per_page=100&page=1&state=active", mockClient.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/milestones?per_page=100&page=2&state=active", mockClient.Requests[1].URL.String())

		require.Len(t, milestones, 2)
		assert.Equal(t, "v1.0", milestones[0].Title)
		assert.Equal(t, "v2.0", milestones[1].Title)
	})

	t.Run("error", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"error": "not found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, err := client.ListMilestones("456")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 404")
	})
}
