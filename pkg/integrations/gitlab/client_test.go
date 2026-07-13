package gitlab

import (
	"context"
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

func Test__Client__GetIssue(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusOK, `{
					"id": 41,
					"iid": 7,
					"project_id": 456,
					"title": "Login page rendering issue",
					"state": "opened",
					"labels": ["bug"],
					"web_url": "https://gitlab.com/group/project/-/issues/7"
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

		issue, err := client.GetIssue("456", "7")
		require.NoError(t, err)
		require.NotNil(t, issue)
		assert.Equal(t, 7, issue.IID)
		assert.Equal(t, "Login page rendering issue", issue.Title)

		require.Len(t, mockClient.Requests, 1)
		assert.Equal(t, http.MethodGet, mockClient.Requests[0].Method)
		assert.Equal(t, "https://gitlab.com/api/v4/projects/456/issues/7", mockClient.Requests[0].URL.String())
		assert.Equal(t, "token", mockClient.Requests[0].Header.Get("PRIVATE-TOKEN"))
	})

	t.Run("not found", func(t *testing.T) {
		mockClient := &contexts.HTTPContext{
			Responses: []*http.Response{
				GitlabMockResponse(http.StatusNotFound, `{"message": "404 Not found"}`),
			},
		}

		client := &Client{
			baseURL:    "https://gitlab.com",
			token:      "token",
			authType:   AuthTypePersonalAccessToken,
			groupID:    "123",
			httpClient: mockClient,
		}

		_, err := client.GetIssue("456", "999")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get issue")
	})
}
