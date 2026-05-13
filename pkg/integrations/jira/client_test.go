package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// newAuthorizedIntegration returns an IntegrationContext that has the
// OAuth access token + cloud metadata already populated, simulating a
// successfully-authorized integration.
func newAuthorizedIntegration() *contexts.IntegrationContext {
	ctx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"clientId":     "client-id",
			"clientSecret": "client-secret",
		},
		Metadata: Metadata{
			CloudID: "cloud-123",
			SiteURL: "https://your-domain.atlassian.net",
		},
	}
	_ = ctx.SetSecret(OAuthAccessToken, []byte("test-access-token"))
	return ctx
}

func newAuthorizedIntegrationWithMetadata(metadata Metadata) *contexts.IntegrationContext {
	if metadata.CloudID == "" {
		metadata.CloudID = "cloud-123"
	}
	ctx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"clientId":     "client-id",
			"clientSecret": "client-secret",
		},
		Metadata: metadata,
	}
	_ = ctx.SetSecret(OAuthAccessToken, []byte("test-access-token"))
	return ctx
}

func Test__NewClient(t *testing.T) {
	t.Run("missing access token -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
			Metadata: Metadata{CloudID: "cloud-123"},
		}

		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, appCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing access token")
	})

	t.Run("missing cloud ID -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"clientId":     "client-id",
				"clientSecret": "client-secret",
			},
		}
		_ = appCtx.SetSecret(OAuthAccessToken, []byte("test-token"))

		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, appCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing Jira cloud ID")
	})

	t.Run("successful client creation", func(t *testing.T) {
		appCtx := newAuthorizedIntegration()
		client, err := NewClient(&contexts.HTTPContext{}, appCtx)

		require.NoError(t, err)
		assert.Equal(t, "test-access-token", client.Token)
		assert.Equal(t, "cloud-123", client.CloudID)
	})
}

func Test__Client__GetCurrentUser(t *testing.T) {
	t.Run("successful get current user", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"accountId":"123","displayName":"Test User","emailAddress":"test@example.com"}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		user, err := client.GetCurrentUser()

		require.NoError(t, err)
		assert.Equal(t, "123", user.AccountID)
		assert.Equal(t, "Test User", user.DisplayName)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ex/jira/cloud-123/rest/api/3/myself")
		assert.Equal(t, "Bearer test-access-token", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("auth failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
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
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"10000","key":"TEST","name":"Test Project"},{"id":"10001","key":"DEMO","name":"Demo Project"}]`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		projects, err := client.ListProjects()

		require.NoError(t, err)
		require.Len(t, projects, 2)
		assert.Equal(t, "TEST", projects[0].Key)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ex/jira/cloud-123/rest/api/3/project")
	})
}

func Test__Client__GetIssue(t *testing.T) {
	t.Run("successful get issue", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-123","fields":{"summary":"Test issue"}}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		issue, err := client.GetIssue("TEST-123")

		require.NoError(t, err)
		assert.Equal(t, "10001", issue.ID)
		assert.Equal(t, "TEST-123", issue.Key)
		assert.Equal(t, "Test issue", issue.Fields["summary"])
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ex/jira/cloud-123/rest/api/3/issue/TEST-123")
	})

	t.Run("issue not found -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Issue does not exist"]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
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
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10002","key":"TEST-124"}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		response, err := client.CreateIssue(&CreateIssueRequest{
			Fields: CreateIssueFields{
				Project:   ProjectRef{Key: "TEST"},
				IssueType: IssueType{Name: "Task"},
				Summary:   "New test issue",
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "10002", response.ID)
		assert.Equal(t, "TEST-124", response.Key)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ex/jira/cloud-123/rest/api/3/issue")
	})

	t.Run("issue creation failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Project is required"]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.CreateIssue(&CreateIssueRequest{Fields: CreateIssueFields{}})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}

func Test__Client__UpdateIssue(t *testing.T) {
	t.Run("successful update returns no error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = client.UpdateIssue("TEST-1", &UpdateIssueRequest{
			Fields: map[string]any{"summary": "new"},
		}, UpdateIssueOptions{})

		require.NoError(t, err)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ex/jira/cloud-123/rest/api/3/issue/TEST-1")
	})

	t.Run("update error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["bad"]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = client.UpdateIssue("TEST-1", &UpdateIssueRequest{Fields: map[string]any{}}, UpdateIssueOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})
}

func Test__Client__DeleteIssue(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = client.DeleteIssue("TEST-1", DeleteIssueOptions{DeleteSubtasks: true})
		require.NoError(t, err)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "deleteSubtasks=true")
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})

	t.Run("delete error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusForbidden,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["no perm"]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = client.DeleteIssue("TEST-1", DeleteIssueOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "403")
	})
}

func Test__Client__RegisterWebhooks(t *testing.T) {
	t.Run("registers webhook and returns ids", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"webhookRegistrationResult":[{"createdWebhookId":42}]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		ids, err := client.RegisterWebhooks("https://hook.example.com", []WebhookRegistration{
			{Events: []string{JiraEventIssueCreated}, JQLFilter: "project = TEST"},
		})

		require.NoError(t, err)
		assert.Equal(t, []int{42}, ids)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ex/jira/cloud-123/rest/api/3/webhook")
	})

	t.Run("propagates registration errors", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"webhookRegistrationResult":[{"errors":["bad jql"]}]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.RegisterWebhooks("https://hook.example.com", []WebhookRegistration{
			{Events: []string{JiraEventIssueCreated}, JQLFilter: "bad"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad jql")
	})
}

func Test__Client__DeleteWebhooks(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(``))},
		},
	}

	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	err = client.DeleteWebhooks([]int{42, 43})
	require.NoError(t, err)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
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
		assert.Equal(t, "Hello world", result.Content[0].Content[0].Text)
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		result := WrapInADF("")
		assert.Nil(t, result)
	})
}
