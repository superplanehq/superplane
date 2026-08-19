package gitlab

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
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
				"authType":    AuthTypePersonalAccessToken,
				"baseUrl":     "https://gitlab.example.com",
				"accessToken": "pat-123",
				"groupId":     "group-123",
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
			CurrentSecrets: map[string]core.IntegrationSecret{
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

	t.Run("missing groupId is allowed", func(t *testing.T) {
		ctx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"authType":    AuthTypePersonalAccessToken,
				"accessToken": "pat-123",
			},
		}
		client, err := NewClient(mockClient, ctx)
		require.NoError(t, err)
		assert.Equal(t, "", client.groupID)
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

	t.Run("personal projects when no group is configured", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 7, "username": "solo"}`),
				GitlabMockResponse(http.StatusOK, `[{"id": 1, "path_with_namespace": "solo/project1", "web_url": "https://gitlab.com/solo/project1"}]`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "",
			httpClient: mockClient,
		}

		user, projects, err := client.FetchIntegrationData()
		require.NoError(t, err)

		require.Len(t, mockClient.Requests, 2)
		assert.Equal(t, "https://gitlab.com/api/v4/user", mockClient.Requests[0].URL.String())
		assert.Equal(t, "https://gitlab.com/api/v4/users/7/projects?per_page=100&page=1", mockClient.Requests[1].URL.String())

		require.NotNil(t, user)
		assert.Equal(t, 7, user.ID)

		require.Len(t, projects, 1)
		assert.Equal(t, "solo/project1", projects[0].PathWithNamespace)
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

	t.Run("current user fetch fails when no group is configured", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusUnauthorized, `{"message": "401 Unauthorized"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "",
			httpClient: mockClient,
		}

		_, _, err := client.FetchIntegrationData()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get current user")
		require.Len(t, mockClient.Requests, 1)
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

func Test__Client__CreatePipeline(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{
					"id": 12345,
					"iid": 321,
					"project_id": 456,
					"status": "pending",
					"ref": "main",
					"sha": "abc123",
					"web_url": "https://gitlab.com/group/project/-/pipelines/12345"
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

		pipeline, err := client.CreatePipeline(context.Background(), "456", &CreatePipelineRequest{
			Ref: "main",
			Inputs: map[string]string{
				"target_env": "dev",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		assert.Equal(t, 12345, pipeline.ID)
		assert.Equal(t, "pending", pipeline.Status)
		assert.Equal(t, "main", pipeline.Ref)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPost, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/pipeline", mockClient.Requests[0].URL.String())
		assert.Equal(t, "token", mockClient.Requests[0].Header.Get("PRIVATE-TOKEN"))

		body, readErr := io.ReadAll(mockClient.Requests[0].Body)
		require.NoError(t, readErr)
		bodyString := string(body)
		assert.True(t, strings.Contains(bodyString, `"ref":"main"`))
		assert.True(t, strings.Contains(bodyString, `"inputs":{"target_env":"dev"}`))
	})
}

func Test__Client__GetPipeline(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": 12345,
					"iid": 321,
					"project_id": 456,
					"status": "running",
					"ref": "main",
					"sha": "abc123",
					"web_url": "https://gitlab.com/group/project/-/pipelines/12345"
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

		pipeline, err := client.GetPipeline("456", 12345)
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		assert.Equal(t, 12345, pipeline.ID)
		assert.Equal(t, "running", pipeline.Status)
		assert.Equal(t, "main", pipeline.Ref)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/pipelines/12345", mockClient.Requests[0].URL.String())
	})
}

func Test__Client__GetLatestPipeline(t *testing.T) {
	t.Run("success with ref", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": 12346,
					"iid": 322,
					"project_id": 456,
					"status": "success",
					"ref": "main",
					"sha": "def456",
					"web_url": "https://gitlab.com/group/project/-/pipelines/12346"
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

		pipeline, err := client.GetLatestPipeline("456", "main")
		require.NoError(t, err)
		require.NotNil(t, pipeline)
		assert.Equal(t, 12346, pipeline.ID)
		assert.Equal(t, "success", pipeline.Status)
		assert.Equal(t, "main", pipeline.Ref)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/pipelines/latest?ref=main", mockClient.Requests[0].URL.String())
	})
}

func Test__Client__GetPipelineTestReportSummary(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"total": {
						"time": 12.34,
						"count": 40,
						"success": 39,
						"failed": 1,
						"skipped": 0,
						"error": 0
					},
					"test_suites": [
						{
							"name": "rspec",
							"total_time": 12.34,
							"total_count": 40,
							"success_count": 39,
							"failed_count": 1
						}
					]
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

		summary, err := client.GetPipelineTestReportSummary("456", 12345)
		require.NoError(t, err)
		require.NotNil(t, summary)
		assert.Equal(t, 40.0, summary.Total["count"])
		require.Len(t, summary.TestSuites, 1)
		assert.Equal(t, "rspec", summary.TestSuites[0]["name"])

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/pipelines/12345/test_report_summary", mockClient.Requests[0].URL.String())
	})
}

func Test__Client__ListPipelines(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[
					{"id": 1001, "status": "running", "ref": "main"},
					{"id": 1000, "status": "success", "ref": "release/v1.0"}
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

		pipelines, err := client.ListPipelines("456")
		require.NoError(t, err)
		require.Len(t, pipelines, 2)
		assert.Equal(t, 1001, pipelines[0].ID)
		assert.Equal(t, "running", pipelines[0].Status)
		assert.Equal(t, "main", pipelines[0].Ref)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/pipelines?per_page=100&page=1", mockClient.Requests[0].URL.String())
	})
}

func Test__Client__CreateMergeRequestAwardEmoji(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{"id": 25, "name": "eyes", "user": {"id": 42}}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		result, err := client.CreateMergeRequestAwardEmoji(context.Background(), "1", "1", &CreateAwardEmojiRequest{Name: "eyes"})
		require.NoError(t, err)
		assert.Equal(t, 25, result.ID)
		assert.Equal(t, "eyes", result.Name)

		require.Len(t, mockClient.Requests, 1)
	})

	t.Run("already exists - returns the existing award emoji", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message":"404 Award Emoji Name has already been taken"}`),
				GitlabMockResponse(http.StatusOK, `{"id": 42}`),
				GitlabMockResponse(http.StatusOK, `[
					{"id": 24, "name": "eyes", "user": {"id": 7}},
					{"id": 25, "name": "eyes", "user": {"id": 42}},
					{"id": 26, "name": "rocket", "user": {"id": 42}}
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

		result, err := client.CreateMergeRequestAwardEmoji(context.Background(), "1", "1", &CreateAwardEmojiRequest{Name: "eyes"})
		require.NoError(t, err)
		assert.Equal(t, 25, result.ID)
		assert.Equal(t, "eyes", result.Name)
		assert.Equal(t, 42, result.User.ID)

		require.Len(t, mockClient.Requests, 3)
		assert.Equal(t, http.MethodPost, mockClient.Requests[0].Method)
		assert.Equal(t, http.MethodGet, mockClient.Requests[1].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/user", mockClient.Requests[1].URL.String())
		assert.Equal(t, http.MethodGet, mockClient.Requests[2].Method)
	})

	t.Run("already exists but not found in listing - returns an error", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message":"404 Award Emoji Name has already been taken"}`),
				GitlabMockResponse(http.StatusOK, `{"id": 42}`),
				GitlabMockResponse(http.StatusOK, `[]`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, err := client.CreateMergeRequestAwardEmoji(context.Background(), "1", "1", &CreateAwardEmojiRequest{Name: "eyes"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reported as already existing but could not be found")
	})

	t.Run("other error - not treated as already-exists", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message":"404 Project Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, err := client.CreateMergeRequestAwardEmoji(context.Background(), "1", "1", &CreateAwardEmojiRequest{Name: "eyes"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create award emoji")

		require.Len(t, mockClient.Requests, 1)
	})
}

func Test__Client__AcceptMergeRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": 1,
					"iid": 42,
					"project_id": 456,
					"title": "feat: add login page",
					"state": "merged",
					"merged_at": "2026-02-13T11:16:17.520Z",
					"source_branch": "feature/login-page",
					"target_branch": "main",
					"merge_commit_sha": "9999999999999999999999999999999999999999",
					"web_url": "https://gitlab.com/group/project/-/merge_requests/42"
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

		squash := true
		removeSourceBranch := true
		mergeRequest, err := client.AcceptMergeRequest(context.Background(), "456", "42", &AcceptMergeRequestRequest{
			MergeCommitMessage:       "Merge login page",
			Squash:                   &squash,
			SquashCommitMessage:      "feat: add login page",
			ShouldRemoveSourceBranch: &removeSourceBranch,
			SHA:                      "8888888888888888888888888888888888888888",
		})
		require.NoError(t, err)
		require.NotNil(t, mergeRequest)
		assert.Equal(t, 42, mergeRequest.IID)
		assert.Equal(t, "merged", mergeRequest.State)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPut, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/merge_requests/42/merge", mockClient.Requests[0].URL.String())
		assert.Equal(t, "token", mockClient.Requests[0].Header.Get("PRIVATE-TOKEN"))

		body, readErr := io.ReadAll(mockClient.Requests[0].Body)
		require.NoError(t, readErr)
		bodyString := string(body)
		assert.True(t, strings.Contains(bodyString, `"merge_commit_message":"Merge login page"`))
		assert.True(t, strings.Contains(bodyString, `"squash":true`))
		assert.True(t, strings.Contains(bodyString, `"squash_commit_message":"feat: add login page"`))
		assert.True(t, strings.Contains(bodyString, `"should_remove_source_branch":true`))
		assert.True(t, strings.Contains(bodyString, `"sha":"8888888888888888888888888888888888888888"`))
	})

	t.Run("omits optional fields when unset", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 1, "iid": 42, "state": "merged"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, err := client.AcceptMergeRequest(context.Background(), "456", "42", &AcceptMergeRequestRequest{})
		require.NoError(t, err)

		require.Len(t, mockClient.Requests, 1)
		body, readErr := io.ReadAll(mockClient.Requests[0].Body)
		require.NoError(t, readErr)
		assert.Equal(t, "{}", string(body))
	})

	t.Run("conflict surfaces GitLab's message", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusConflict, `{"message": "Merge request is not mergeable"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, err := client.AcceptMergeRequest(context.Background(), "456", "42", &AcceptMergeRequestRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Merge request is not mergeable")
	})

	t.Run("conflict with empty body falls back to SHA mismatch message", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusConflict, ``),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, err := client.AcceptMergeRequest(context.Background(), "456", "42", &AcceptMergeRequestRequest{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SHA does not match HEAD of source branch")
	})
}

func Test__Client__ApproveMergeRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{
					"id": 1,
					"iid": 42,
					"project_id": 456,
					"title": "feat: add login page",
					"state": "opened",
					"approvals_required": 2,
					"approvals_left": 1,
					"approved_by": [
						{
							"user": {"id": 1, "name": "Administrator", "username": "root"},
							"approved_at": "2026-02-13T10:15:30.000Z"
						}
					]
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

		approval, err := client.ApproveMergeRequest(context.Background(), "456", "42", &ApproveMergeRequestRequest{
			SHA: "8888888888888888888888888888888888888888",
		})
		require.NoError(t, err)
		require.NotNil(t, approval)
		assert.Equal(t, 42, approval.IID)
		assert.Equal(t, 1, approval.ApprovalsLeft)
		require.Len(t, approval.ApprovedBy, 1)
		assert.Equal(t, "root", approval.ApprovedBy[0].User.Username)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPost, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/merge_requests/42/approve", mockClient.Requests[0].URL.String())
		assert.Equal(t, "token", mockClient.Requests[0].Header.Get("PRIVATE-TOKEN"))

		body, readErr := io.ReadAll(mockClient.Requests[0].Body)
		require.NoError(t, readErr)
		assert.True(t, strings.Contains(string(body), `"sha":"8888888888888888888888888888888888888888"`))
	})

	t.Run("SHA mismatch", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusConflict, `{"message": "SHA does not match HEAD of source branch"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, err := client.ApproveMergeRequest(context.Background(), "456", "42", &ApproveMergeRequestRequest{
			SHA: "0000000000000000000000000000000000000000",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SHA does not match HEAD of source branch")
	})
}

func Test__Client__GetMergeRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 1, "iid": 42, "title": "Test MR", "state": "opened", "draft": true}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.GetMergeRequest(context.Background(), "456", "42")
		require.NoError(t, err)
		assert.Equal(t, 42, result.IID)
		assert.Equal(t, "Test MR", result.Title)
		assert.True(t, result.Draft)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/merge_requests/42", mockClient.Requests[0].URL.String())
	})

	t.Run("not found", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Merge Request Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.GetMergeRequest(context.Background(), "456", "999")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get merge request")
	})
}

func Test__Client__ListEnvironments(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[
					{"id": 1, "name": "production", "slug": "production", "external_url": "https://prod.example.com", "state": "available", "tier": "production"},
					{"id": 2, "name": "staging", "slug": "staging", "external_url": "https://staging.example.com", "state": "available", "tier": "staging"}
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

		environments, err := client.ListEnvironments("456")
		require.NoError(t, err)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/environments?per_page=100&page=1&states=available", mockClient.Requests[0].URL.String())
		assert.Equal(t, "token", mockClient.Requests[0].Header.Get("PRIVATE-TOKEN"))

		require.Len(t, environments, 2)
		assert.Equal(t, "production", environments[0].Name)
		assert.Equal(t, "https://prod.example.com", environments[0].ExternalURL)
		assert.Equal(t, "staging", environments[1].Name)
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

		_, err := client.ListEnvironments("456")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 404")
	})
}

func Test__Client__CreateDeployment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{
					"id": 42,
					"iid": 2,
					"ref": "main",
					"sha": "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
					"created_at": "2016-08-11T11:32:35.444Z",
					"status": "running",
					"environment": {"id": 9, "name": "production", "external_url": "https://prod.example.com"},
					"deployable": null
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

		deployment, err := client.CreateDeployment(context.Background(), "456", &CreateDeploymentRequest{
			Environment: "production",
			Ref:         "main",
			SHA:         "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
			Tag:         false,
			Status:      "running",
		})
		require.NoError(t, err)
		require.NotNil(t, deployment)
		assert.Equal(t, 42, deployment.ID)
		assert.Equal(t, "running", deployment.Status)
		require.NotNil(t, deployment.Environment)
		assert.Equal(t, "production", deployment.Environment.Name)
		assert.Equal(t, "https://prod.example.com", deployment.Environment.ExternalURL)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPost, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/deployments", mockClient.Requests[0].URL.String())

		body, readErr := io.ReadAll(mockClient.Requests[0].Body)
		require.NoError(t, readErr)
		bodyString := string(body)
		assert.Contains(t, bodyString, `"environment":"production"`)
		assert.Contains(t, bodyString, `"ref":"main"`)
		assert.Contains(t, bodyString, `"tag":false`)
		assert.Contains(t, bodyString, `"status":"running"`)
	})

	t.Run("error", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusBadRequest, `{"message": "sha is invalid"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, err := client.CreateDeployment(context.Background(), "456", &CreateDeploymentRequest{
			Environment: "production",
			Ref:         "main",
			SHA:         "bad",
			Status:      "running",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create deployment")
	})
}

func Test__Client__UpdateDeployment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": 42,
					"iid": 2,
					"ref": "main",
					"sha": "a91957a858320c0e17f3a0eca7cfacbff50ea29a",
					"created_at": "2016-08-11T11:32:35.444Z",
					"updated_at": "2016-08-11T11:34:00.000Z",
					"status": "success",
					"environment": {"id": 9, "name": "production", "external_url": "https://prod.example.com"}
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

		deployment, err := client.UpdateDeployment(context.Background(), "456", 42, &UpdateDeploymentRequest{
			Status: "success",
		})
		require.NoError(t, err)
		require.NotNil(t, deployment)
		assert.Equal(t, 42, deployment.ID)
		assert.Equal(t, "success", deployment.Status)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPut, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/deployments/42", mockClient.Requests[0].URL.String())

		body, readErr := io.ReadAll(mockClient.Requests[0].Body)
		require.NoError(t, readErr)
		assert.Contains(t, string(body), `"status":"success"`)
	})

	t.Run("error", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Deployment Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, err := client.UpdateDeployment(context.Background(), "456", 99, &UpdateDeploymentRequest{Status: "failed"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update deployment")
	})
}

func Test__Client__GetIssue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 101, "iid": 1, "title": "Test Issue", "state": "opened"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.GetIssue(context.Background(), "1", "1")
		require.NoError(t, err)
		assert.Equal(t, 101, result.ID)
		assert.Equal(t, "Test Issue", result.Title)
		assert.Equal(t, "opened", result.State)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/issues/1", mockClient.Requests[0].URL.String())
	})

	t.Run("not found", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Issue Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.GetIssue(context.Background(), "1", "999")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get issue")
	})
}

func Test__Client__UpdateIssue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 101, "iid": 1, "title": "Updated Title", "state": "closed"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		title, stateEvent := "Updated Title", "close"
		req := &UpdateIssueRequest{Title: &title, StateEvent: &stateEvent}
		result, err := client.UpdateIssue(context.Background(), "1", "1", req)

		require.NoError(t, err)
		assert.Equal(t, "Updated Title", result.Title)
		assert.Equal(t, "closed", result.State)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPut, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/issues/1", mockClient.Requests[0].URL.String())

		body, _ := io.ReadAll(mockClient.Requests[0].Body)
		assert.True(t, strings.Contains(string(body), `"state_event":"close"`))
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusBadRequest, `{"message": "400 Bad Request"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		title := "x"
		_, err := client.UpdateIssue(context.Background(), "1", "1", &UpdateIssueRequest{Title: &title})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update issue")
	})
}

func Test__Client__CreateIssueNote(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{"id": 55, "body": "Great work!", "noteable_type": "Issue"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.CreateIssueNote(context.Background(), "1", "5", &CreateNoteRequest{Body: "Great work!"})

		require.NoError(t, err)
		assert.Equal(t, 55, result.ID)
		assert.Equal(t, "Great work!", result.Body)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPost, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/issues/5/notes", mockClient.Requests[0].URL.String())
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Issue Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.CreateIssueNote(context.Background(), "1", "999", &CreateNoteRequest{Body: "x"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create issue note")
	})

	t.Run("quick-action-only body returns 202 Accepted", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusAccepted, `{"commands_changes":{"state_event":"close"},"summary":["Closed this issue."]}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.CreateIssueNote(context.Background(), "1", "5", &CreateNoteRequest{Body: "/close"})
		require.NoError(t, err)
		assert.Equal(t, "Closed this issue.", result.Body)
		assert.Equal(t, 0, result.ID)
	})
}

func Test__Client__CreateMergeRequestNote(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{"id": 55, "body": "Great work!", "noteable_type": "MergeRequest"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.CreateMergeRequestNote(context.Background(), "1", "5", &CreateNoteRequest{Body: "Great work!"})

		require.NoError(t, err)
		assert.Equal(t, 55, result.ID)
		assert.Equal(t, "Great work!", result.Body)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPost, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/merge_requests/5/notes", mockClient.Requests[0].URL.String())
	})

	// GitLab confirmed live: a note whose body is only the "/ready" quick
	// action returns 202 with a commands summary, not a Note object.
	t.Run("quick-action-only body returns 202 Accepted", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusAccepted, `{"commands_changes":{"wip_event":"ready"},"summary":["Marked this merge request as ready."]}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.CreateMergeRequestNote(context.Background(), "1", "5", &CreateNoteRequest{Body: "/ready"})
		require.NoError(t, err)
		assert.Equal(t, "Marked this merge request as ready.", result.Body)
		assert.Equal(t, 0, result.ID)
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Merge Request Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.CreateMergeRequestNote(context.Background(), "1", "999", &CreateNoteRequest{Body: "x"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create merge request note")
	})
}

func Test__Client__CreateRelease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{
					"tag_name": "v1.0.0",
					"name": "Release 1.0.0",
					"description": "First release",
					"released_at": "2026-01-03T01:56:19.539Z",
					"author": {"id": 1, "username": "root"},
					"milestones": [{"id": 51, "title": "v1.0"}],
					"assets": {"count": 2, "sources": [{"format": "zip", "url": "https://example.com/v1.0.0.zip"}]},
					"_links": {"self": "https://gitlab.com/root/example/-/releases/v1.0.0"}
				}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		req := &CreateReleaseRequest{TagName: "v1.0.0", Ref: "main", Name: "Release 1.0.0"}
		result, err := client.CreateRelease(context.Background(), "1", req)

		require.NoError(t, err)
		assert.Equal(t, "v1.0.0", result.TagName)
		assert.Equal(t, "Release 1.0.0", result.Name)
		assert.Equal(t, "First release", result.Description)

		require.NotNil(t, result.Author)
		assert.Equal(t, "root", result.Author.Username)

		require.Len(t, result.Milestones, 1)
		assert.Equal(t, "v1.0", result.Milestones[0].Title)

		require.NotNil(t, result.Assets)
		assert.Equal(t, 2, result.Assets.Count)
		require.Len(t, result.Assets.Sources, 1)
		assert.Equal(t, "zip", result.Assets.Sources[0].Format)

		require.NotNil(t, result.Links)
		assert.Equal(t, "https://gitlab.com/root/example/-/releases/v1.0.0", result.Links.Self)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPost, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/releases", mockClient.Requests[0].URL.String())

		body, _ := io.ReadAll(mockClient.Requests[0].Body)
		assert.True(t, strings.Contains(string(body), `"tag_name":"v1.0.0"`))
		assert.True(t, strings.Contains(string(body), `"ref":"main"`))
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusBadRequest, `{"message": "Tag name has already been taken"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.CreateRelease(context.Background(), "1", &CreateReleaseRequest{TagName: "v1.0.0"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create release")
	})
}

func Test__Client__UpdateRelease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"tag_name": "v1.0.0", "name": "Updated name", "description": "Updated notes"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		name := "Updated name"
		milestones := []string{"v1.0"}
		req := &UpdateReleaseRequest{Name: &name, Milestones: &milestones}
		result, err := client.UpdateRelease(context.Background(), "1", "v1.0.0", req)

		require.NoError(t, err)
		assert.Equal(t, "Updated name", result.Name)
		assert.Equal(t, "Updated notes", result.Description)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPut, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/releases/v1.0.0", mockClient.Requests[0].URL.String())

		body, _ := io.ReadAll(mockClient.Requests[0].Body)
		assert.True(t, strings.Contains(string(body), `"milestones":["v1.0"]`))
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		name := "x"
		_, err := client.UpdateRelease(context.Background(), "1", "v1.0.0", &UpdateReleaseRequest{Name: &name})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update release")
	})
}

func Test__Client__GetRelease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"tag_name": "v1.0.0", "name": "Release 1.0.0"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.GetRelease(context.Background(), "1", "v1.0.0")

		require.NoError(t, err)
		assert.Equal(t, "v1.0.0", result.TagName)
		assert.Equal(t, "Release 1.0.0", result.Name)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/releases/v1.0.0", mockClient.Requests[0].URL.String())
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.GetRelease(context.Background(), "1", "v1.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get release")
	})
}

func Test__Client__DeleteRelease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"tag_name": "v1.0.0", "name": "Release 1.0.0"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.DeleteRelease(context.Background(), "1", "v1.0.0")

		require.NoError(t, err)
		assert.Equal(t, "v1.0.0", result.TagName)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodDelete, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/releases/v1.0.0", mockClient.Requests[0].URL.String())
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.DeleteRelease(context.Background(), "1", "v1.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete release")
	})
}

func Test__Client__DeleteTag(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNoContent, ``),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		err := client.DeleteTag(context.Background(), "1", "v1.0.0")
		require.NoError(t, err)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodDelete, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/repository/tags/v1.0.0", mockClient.Requests[0].URL.String())
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Tag Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		err := client.DeleteTag(context.Background(), "1", "v1.0.0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete tag")
	})
}

func Test__Client__GetLatestRelease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[{"tag_name": "v2.0.0", "name": "Release 2.0.0"}]`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.GetLatestRelease(context.Background(), "1")

		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", result.TagName)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/1/releases?order_by=released_at&sort=desc&per_page=100", mockClient.Requests[0].URL.String())
	})

	t.Run("skips upcoming releases sorted above published ones", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[
					{"tag_name": "v3.0.0-rc1", "upcoming_release": true},
					{"tag_name": "v2.0.0", "upcoming_release": false}
				]`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.GetLatestRelease(context.Background(), "1")

		require.NoError(t, err)
		assert.Equal(t, "v2.0.0", result.TagName)
	})

	t.Run("no published releases found", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `[{"tag_name": "v3.0.0-rc1", "upcoming_release": true}]`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.GetLatestRelease(context.Background(), "1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no published releases found")
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusInternalServerError, `{"message": "internal error"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.GetLatestRelease(context.Background(), "1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list releases")
	})
}

func Test__Client__CreateCommitStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusCreated, `{
					"id": 98,
					"sha": "18f3e63d05582537db6d183d9d557be09e1f90c8",
					"ref": "main",
					"status": "success",
					"name": "ci/superplane",
					"target_url": "https://ci.example.com/runs/42",
					"description": "Build passed",
					"created_at": "2026-02-13T18:00:00.000Z",
					"finished_at": "2026-02-13T18:04:12.000Z",
					"allow_failure": false,
					"coverage": 92.5,
					"author": {"id": 1, "username": "root", "name": "Administrator", "state": "active"}
				}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		status, err := client.CreateCommitStatus(context.Background(), "456", "18f3e63d05582537db6d183d9d557be09e1f90c8", &CreateCommitStatusRequest{
			State:       "success",
			Name:        "ci/superplane",
			TargetURL:   "https://ci.example.com/runs/42",
			Description: "Build passed",
		})
		require.NoError(t, err)
		require.NotNil(t, status)
		assert.Equal(t, 98, status.ID)
		assert.Equal(t, "success", status.Status)
		assert.Equal(t, "ci/superplane", status.Name)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodPost, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/statuses/18f3e63d05582537db6d183d9d557be09e1f90c8", mockClient.Requests[0].URL.String())
		assert.Equal(t, "token", mockClient.Requests[0].Header.Get("PRIVATE-TOKEN"))

		body, readErr := io.ReadAll(mockClient.Requests[0].Body)
		require.NoError(t, readErr)
		bodyString := string(body)
		assert.Contains(t, bodyString, `"state":"success"`)
		assert.Contains(t, bodyString, `"name":"ci/superplane"`)
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusBadRequest, `{"message": "400 Bad Request"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.CreateCommitStatus(context.Background(), "456", "abc123", &CreateCommitStatusRequest{State: "success"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create commit status")
	})
}

func Test__Client__GetCommit(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": "18f3e63d05582537db6d183d9d557be09e1f90c8",
					"short_id": "18f3e63d",
					"title": "feat: add health endpoint",
					"status": "failed",
					"web_url": "https://gitlab.com/g/p/-/commit/18f3e63d05582537db6d183d9d557be09e1f90c8",
					"last_pipeline": {"id": 1024, "ref": "main", "sha": "18f3e63d05582537db6d183d9d557be09e1f90c8", "status": "failed", "web_url": "https://gitlab.com/g/p/-/pipelines/1024"}
				}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		commit, err := client.GetCommit(context.Background(), "456", "main")
		require.NoError(t, err)
		require.NotNil(t, commit)
		assert.Equal(t, "failed", commit.Status)
		require.NotNil(t, commit.LastPipeline)
		assert.Equal(t, "failed", commit.LastPipeline.Status)
		assert.Equal(t, 1024, commit.LastPipeline.ID)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/repository/commits/main", mockClient.Requests[0].URL.String())
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Commit Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.GetCommit(context.Background(), "456", "abc123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get commit")
	})
}

func Test__Client__GetGroup(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"id": 123, "full_path": "my-org/my-subgroup"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		result, err := client.GetGroup("my-org/my-subgroup")

		require.NoError(t, err)
		assert.Equal(t, 123, result.ID)
		assert.Equal(t, "my-org/my-subgroup", result.FullPath)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/groups/my-org%2Fmy-subgroup", mockClient.Requests[0].URL.String())
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Group Not Found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, err := client.GetGroup("123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get group")
	})
}

func Test__Client__GetCiMinutesUsage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"data": {
						"ciMinutesUsage": {
							"nodes": [{"month": "July", "monthIso8601": "2026-07-01", "minutes": 245, "sharedRunnersDuration": 14700}]
						}
					}
				}`),
				GitlabMockResponse(http.StatusOK, `{
					"data": {
						"ciMinutesProjectMonthlyUsage": {
							"nodes": [
								{"minutes": 180, "sharedRunnersDuration": 10800, "project": {"id": "gid://gitlab/Project/1", "name": "hello-world", "fullPath": "felixgateru/hello-world"}}
							],
							"pageInfo": {"hasNextPage": false, "endCursor": ""}
						}
					}
				}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		namespaceGID := "gid://gitlab/Group/123"
		usage, projects, err := client.GetCiMinutesUsage(context.Background(), &namespaceGID, "2026-07-01")

		require.NoError(t, err)
		assert.Equal(t, "July", usage.Month)
		assert.Equal(t, 245, usage.Minutes)

		require.Len(t, projects, 1)
		assert.Equal(t, "hello-world", projects[0].Project.Name)

		require.Len(t, mockClient.Requests, 2)
		for _, req := range mockClient.Requests {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "https://gitlab.com/api/graphql", req.URL.String())
		}

		body, _ := io.ReadAll(mockClient.Requests[0].Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		variables := reqBody["variables"].(map[string]any)
		assert.Equal(t, namespaceGID, variables["namespaceId"])
		assert.Equal(t, "2026-07-01", variables["date"])
	})

	t.Run("pages through more than one page of project usage", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"data": {"ciMinutesUsage": {"nodes": [{"month": "July"}]}}}`),
				GitlabMockResponse(http.StatusOK, `{
					"data": {
						"ciMinutesProjectMonthlyUsage": {
							"nodes": [{"minutes": 100, "project": {"id": "gid://gitlab/Project/1", "name": "project-1"}}],
							"pageInfo": {"hasNextPage": true, "endCursor": "cursor-1"}
						}
					}
				}`),
				GitlabMockResponse(http.StatusOK, `{
					"data": {
						"ciMinutesProjectMonthlyUsage": {
							"nodes": [{"minutes": 50, "project": {"id": "gid://gitlab/Project/2", "name": "project-2"}}],
							"pageInfo": {"hasNextPage": false, "endCursor": ""}
						}
					}
				}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, projects, err := client.GetCiMinutesUsage(context.Background(), nil, "2026-07-01")
		require.NoError(t, err)

		require.Len(t, projects, 2, "must follow pageInfo.hasNextPage and return every page, not just the first 100")
		assert.Equal(t, "project-1", projects[0].Project.Name)
		assert.Equal(t, "project-2", projects[1].Project.Name)

		require.Len(t, mockClient.Requests, 3)
		body, _ := io.ReadAll(mockClient.Requests[2].Body)
		var reqBody map[string]any
		json.Unmarshal(body, &reqBody)
		variables := reqBody["variables"].(map[string]any)
		assert.Equal(t, "cursor-1", variables["after"], "the second page request must pass the previous page's endCursor")
	})

	t.Run("omits namespaceId variable for personal namespace", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"data": {"ciMinutesUsage": {"nodes": []}}}`),
				GitlabMockResponse(http.StatusOK, `{"data": {"ciMinutesProjectMonthlyUsage": {"nodes": [], "pageInfo": {"hasNextPage": false, "endCursor": ""}}}}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, _, err := client.GetCiMinutesUsage(context.Background(), nil, "2026-07-01")
		require.NoError(t, err)

		for _, req := range mockClient.Requests {
			body, _ := io.ReadAll(req.Body)
			var reqBody map[string]any
			json.Unmarshal(body, &reqBody)
			variables := reqBody["variables"].(map[string]any)
			assert.NotContains(t, variables, "namespaceId")
		}
	})

	t.Run("graphql error", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"errors": [{"message": "not authorized to read usage"}]}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, _, err := client.GetCiMinutesUsage(context.Background(), nil, "2026-07-01")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not authorized to read usage")
	})

	t.Run("failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusInternalServerError, `{"message": "internal error"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, _, err := client.GetCiMinutesUsage(context.Background(), nil, "2026-07-01")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "graphql request failed")
	})

	t.Run("project usage page failure", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{"data": {"ciMinutesUsage": {"nodes": [{"month": "July"}]}}}`),
				GitlabMockResponse(http.StatusInternalServerError, `{"message": "internal error"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			httpClient: mockClient,
		}

		_, _, err := client.GetCiMinutesUsage(context.Background(), nil, "2026-07-01")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "graphql request failed")
	})
}
