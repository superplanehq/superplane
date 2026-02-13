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
