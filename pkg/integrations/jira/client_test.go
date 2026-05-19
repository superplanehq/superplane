package jira

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__NewClient(t *testing.T) {
	t.Run("missing access token -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, &contexts.IntegrationContext{
			Metadata: Metadata{CloudID: "cloud-123"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "OAuth accessToken not found")
	})

	t.Run("missing cloud ID -> error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("access-token")},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cloud ID is missing")
	})

	t.Run("successful oauth client creation", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		client, err := NewClient(httpCtx, oauthIntegrationContext())

		require.NoError(t, err)
		assert.Equal(t, "https://api.atlassian.com/ex/jira/cloud-123", client.BaseURL)
		assert.Equal(t, AuthTypeOAuth, client.AuthType)
		assert.Equal(t, "access-token", client.Token)
	})
}

func Test__Client__GetCurrentUser(t *testing.T) {
	t.Run("successful get current user", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"accountId":"123","displayName":"Test User","emailAddress":"test@example.com"}`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		user, err := client.GetCurrentUser()

		require.NoError(t, err)
		assert.Equal(t, "123", user.AccountID)
		assert.Equal(t, "Test User", user.DisplayName)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/myself")
		assert.Equal(t, "Bearer access-token", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("auth failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusUnauthorized, `{"message":"unauthorized"}`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		_, err = client.GetCurrentUser()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})
}

func Test__Client__ListProjects(t *testing.T) {
	t.Run("successful list projects", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `[{"id":"10000","key":"TEST","name":"Test Project"},{"id":"10001","key":"DEMO","name":"Demo Project"}]`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		projects, err := client.ListProjects()

		require.NoError(t, err)
		require.Len(t, projects, 2)
		assert.Equal(t, "TEST", projects[0].Key)
		assert.Equal(t, "DEMO", projects[1].Key)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/project")
	})

	t.Run("empty projects list", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `[]`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		projects, err := client.ListProjects()

		require.NoError(t, err)
		assert.Len(t, projects, 0)
	})
}

func Test__Client__GetIssue(t *testing.T) {
	t.Run("successful get issue", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"id":"10001","key":"TEST-123","self":"https://test.atlassian.net/rest/api/3/issue/10001","fields":{"summary":"Test issue"}}`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		issue, err := client.GetIssue("TEST-123")

		require.NoError(t, err)
		assert.Equal(t, "10001", issue.ID)
		assert.Equal(t, "TEST-123", issue.Key)
		assert.Equal(t, "Test issue", issue.Fields["summary"])
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/issue/TEST-123")
	})

	t.Run("issue not found -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusNotFound, `{"errorMessages":["Issue does not exist"]}`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		_, err = client.GetIssue("INVALID-999")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}

func Test__Client__CreateIssue(t *testing.T) {
	t.Run("successful issue creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusCreated, `{"id":"10002","key":"TEST-124","self":"https://test.atlassian.net/rest/api/3/issue/10002"}`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		req := &CreateIssueRequest{
			Fields: CreateIssueFields{
				Project:   ProjectRef{Key: "TEST"},
				IssueType: IssueType{Name: "Task"},
				Summary:   "New test issue",
			},
		}

		response, err := client.CreateIssue(req)

		require.NoError(t, err)
		assert.Equal(t, "10002", response.ID)
		assert.Equal(t, "TEST-124", response.Key)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/issue")
	})

	t.Run("issue creation with description", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusCreated, `{"id":"10003","key":"TEST-125","self":"https://test.atlassian.net/rest/api/3/issue/10003"}`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		req := &CreateIssueRequest{
			Fields: CreateIssueFields{
				Project:     ProjectRef{Key: "TEST"},
				IssueType:   IssueType{Name: "Bug"},
				Summary:     "Bug report",
				Description: WrapInADF("This is a bug description"),
			},
		}

		response, err := client.CreateIssue(req)

		require.NoError(t, err)
		assert.Equal(t, "TEST-125", response.Key)
	})

	t.Run("issue creation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusBadRequest, `{"errorMessages":["Project is required"]}`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		req := &CreateIssueRequest{
			Fields: CreateIssueFields{},
		}

		_, err = client.CreateIssue(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}

func Test__Client__ListWebhooks(t *testing.T) {
	t.Run("paginated list collects all values", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"isLast":false,"values":[{"id":1},{"id":2}]}`),
				response(http.StatusOK, `{"isLast":true,"values":[{"id":3}]}`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		webhooks, err := client.ListWebhooks()

		require.NoError(t, err)
		require.Len(t, webhooks, 3)
		assert.Equal(t, int64(1), webhooks[0].ID)
		assert.Equal(t, int64(3), webhooks[2].ID)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "startAt=0")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "startAt=2")
	})

	t.Run("empty response stops pagination", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"isLast":false,"values":[]}`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		webhooks, err := client.ListWebhooks()

		require.NoError(t, err)
		assert.Empty(t, webhooks)
	})
}

func Test__Client__RefreshWebhooks(t *testing.T) {
	t.Run("empty IDs -> noop", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		response, err := client.RefreshWebhooks(nil)
		require.NoError(t, err)
		assert.Nil(t, response)
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("sends webhook IDs and parses response", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				response(http.StatusOK, `{"expirationDate":"2030-02-01T00:00:00.000+0000"}`),
			},
		}

		client, err := NewClient(httpContext, oauthIntegrationContext())
		require.NoError(t, err)

		response, err := client.RefreshWebhooks([]int64{111, 222})
		require.NoError(t, err)
		require.NotNil(t, response)
		assert.Equal(t, "2030-02-01T00:00:00.000+0000", response.ExpirationDate)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPut, req.Method)
		assert.Contains(t, req.URL.String(), "/rest/api/3/webhook/refresh")
	})
}

func Test__WrapInADF(t *testing.T) {
	t.Run("wraps text in ADF format", func(t *testing.T) {
		result := WrapInADF("Hello world")

		require.NotNil(t, result)
		assert.Equal(t, "doc", result.Type)
		assert.Equal(t, 1, result.Version)
		require.Len(t, result.Content, 1)
		assert.Equal(t, "paragraph", result.Content[0].Type)
		require.Len(t, result.Content[0].Content, 1)
		assert.Equal(t, "text", result.Content[0].Content[0].Type)
		assert.Equal(t, "Hello world", result.Content[0].Content[0].Text)
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		result := WrapInADF("")
		assert.Nil(t, result)
	})
}

func oauthIntegrationContext() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Metadata: Metadata{
			AuthType: AuthTypeOAuth,
			CloudID:  "cloud-123",
		},
		CurrentSecrets: map[string]core.IntegrationSecret{
			OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("access-token")},
		},
	}
}
