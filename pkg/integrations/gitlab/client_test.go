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
				GitlabMockResponse(http.StatusCreated, `{"id": 101, "iid": 1, "title": "Test Issue", "web_url": "https://gitlab.com/group/project/issues/1"}`),
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

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPost, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/issues", mockClient.Requests[0].URL.String())
	})
}
