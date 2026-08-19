package jira

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const (
	testSiteURL      = "https://your-domain.atlassian.net"
	testCloudID      = "35273b54-3f06-40d2-880f-dd28cf6daafa"
	testAccessToken  = "test-access-token"
	testRefreshToken = "test-refresh-token"
)

// newAuthorizedIntegration returns an IntegrationContext simulating a successfully-connected OAuth integration.
func newAuthorizedIntegration() *contexts.IntegrationContext {
	return newAuthorizedIntegrationWithMetadata(Metadata{
		CloudID: testCloudID,
		SiteURL: testSiteURL,
	})
}

func newAuthorizedIntegrationWithMetadata(metadata Metadata) *contexts.IntegrationContext {
	if metadata.CloudID == "" {
		metadata.CloudID = testCloudID
	}
	if metadata.AccessTokenExpiresAt == "" {
		// Far enough out that refreshAccessToken treats it as valid and skips refreshing,
		// unless a test explicitly sets an expiration to exercise that path.
		metadata.AccessTokenExpiresAt = time.Now().Add(time.Hour).Format(time.RFC3339)
	}

	return &contexts.IntegrationContext{
		CurrentSecrets: map[string]core.IntegrationSecret{
			SecretOAuthAccessToken:  {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)},
			SecretOAuthRefreshToken: {Name: SecretOAuthRefreshToken, Value: []byte(testRefreshToken)},
		},
		Metadata: metadata,
	}
}

// testProxyURL builds the expected OAuth API proxy URL for a REST path, mirroring Client.apiURL.
func testProxyURL(path string) string {
	return APIProxyHost + "/" + testCloudID + path
}

func Test__NewClient(t *testing.T) {
	t.Run("missing cloud id -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			CurrentSecrets: map[string]core.IntegrationSecret{
				SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)},
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cloud id")
	})

	t.Run("missing access token -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Metadata: Metadata{CloudID: testCloudID},
		}

		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "access token")
	})

	t.Run("successful client creation", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, newAuthorizedIntegration())

		require.NoError(t, err)
		assert.Equal(t, testCloudID, client.CloudID)
		assert.Equal(t, testAccessToken, client.AccessToken)
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testProxyURL("/rest/api/3/myself"))
		assert.Equal(t, "Bearer "+testAccessToken, httpContext.Requests[0].Header.Get("Authorization"))
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

	t.Run("401 self-heals via a reactive refresh and retries the request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"refreshed-access","refresh_token":"refreshed-refresh","expires_in":3600}`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accountId":"123","displayName":"Test User"}`))},
			},
		}

		appCtx := newAuthorizedIntegration()
		appCtx.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		user, err := client.GetCurrentUser()
		require.NoError(t, err)
		assert.Equal(t, "123", user.AccountID)

		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, TokenURL, httpContext.Requests[1].URL.String())

		accessToken, _ := findSecret(appCtx, SecretOAuthAccessToken)
		assert.Equal(t, "refreshed-access", accessToken)
	})

	// Regression test: Atlassian refresh tokens are single-use, so a concurrent Sync or request
	// can rotate the stored pair first and make this call's own refresh attempt fail with
	// invalid_grant even though the connection is fine. Adopt the token the winner just stored
	// instead of failing outright.
	t.Run("401 whose own refresh loses a concurrent rotation adopts the winner's token", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`))},
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"accountId":"123","displayName":"Test User"}`))},
			},
		}

		appCtx := newAuthorizedIntegration()
		appCtx.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		// Simulate a concurrent caller having already refreshed and stored a new access token.
		appCtx.CurrentSecrets[SecretOAuthAccessToken] = core.IntegrationSecret{
			Name: SecretOAuthAccessToken, Value: []byte("winner-access"),
		}

		user, err := client.GetCurrentUser()
		require.NoError(t, err)
		assert.Equal(t, "123", user.AccountID)
		assert.Equal(t, "winner-access", client.AccessToken)

		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, "Bearer winner-access", httpContext.Requests[2].Header.Get("Authorization"))
	})

	t.Run("401 whose own refresh fails with no concurrent rotation still errors", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`))},
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))},
			},
		}

		appCtx := newAuthorizedIntegration()
		appCtx.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		_, err = client.GetCurrentUser()
		require.ErrorContains(t, err, "token refresh failed")
	})
}

func Test__Client__Refresh(t *testing.T) {
	t.Run("stores the refreshed token pair and its expiration", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`,
				))},
			},
		}

		appCtx := newAuthorizedIntegration()
		appCtx.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		require.NoError(t, client.Refresh())
		assert.Equal(t, "new-access", client.AccessToken)

		accessToken, _ := findSecret(appCtx, SecretOAuthAccessToken)
		refreshToken, _ := findSecret(appCtx, SecretOAuthRefreshToken)
		assert.Equal(t, "new-access", accessToken)
		assert.Equal(t, "new-refresh", refreshToken)

		remaining, known := accessTokenValidity(appCtx)
		require.True(t, known)
		assert.WithinDuration(t, time.Now().Add(time.Hour), time.Now().Add(remaining), time.Minute)
	})

	t.Run("missing refresh token -> error", func(t *testing.T) {
		appCtx := newAuthorizedIntegration()
		appCtx.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}
		delete(appCtx.CurrentSecrets, SecretOAuthRefreshToken)

		client, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.NoError(t, err)

		require.ErrorContains(t, client.Refresh(), "missing Jira OAuth refresh token")
	})

	// Regression test: Atlassian's refresh tokens are single-use, so the one just sent is already
	// dead the moment this call succeeds. A response that omits the replacement leaves no usable
	// refresh token going forward - fail loudly instead of silently keeping the (now dead) old one
	// and only discovering the break the next time a refresh is attempted.
	t.Run("response omits the new refresh token -> error, old token untouched", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"new-access","expires_in":3600}`,
				))},
			},
		}

		appCtx := newAuthorizedIntegration()
		appCtx.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		require.ErrorContains(t, client.Refresh(), "did not include a new refresh token")

		accessToken, _ := findSecret(appCtx, SecretOAuthAccessToken)
		refreshToken, _ := findSecret(appCtx, SecretOAuthRefreshToken)
		assert.Equal(t, testAccessToken, accessToken, "must not store the new access token without a paired refresh token")
		assert.Equal(t, testRefreshToken, refreshToken)
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testProxyURL("/rest/api/3/project"))
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testProxyURL("/rest/api/3/issue/TEST-123"))
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testProxyURL("/rest/api/3/issue"))
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testProxyURL("/rest/api/3/issue/TEST-1"))
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

func Test__Client__CreateIssueWebhook(t *testing.T) {
	t.Run("wrapped response shape (documented, confirmed live)", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"webhookRegistrationResult":[{"createdWebhookId":1000}]}`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		id, err := client.CreateIssueWebhook("https://example.com/webhook", `project = "ENG"`, []string{"jira:issue_created"})
		require.NoError(t, err)
		assert.Equal(t, int64(1000), id)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/webhook")
	})

	t.Run("bare array response shape (fallback)", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":1001}]`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		id, err := client.CreateIssueWebhook("https://example.com/webhook", `project = "ENG"`, []string{"jira:issue_created"})
		require.NoError(t, err)
		assert.Equal(t, int64(1001), id)
	})

	t.Run("per-webhook error is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"webhookRegistrationResult":[{"errors":["The clause myClause is unsupported"]}]}`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.CreateIssueWebhook("https://example.com/webhook", "bogus", []string{"jira:issue_created"})
		require.ErrorContains(t, err, "myClause is unsupported")
	})

	t.Run("unrecognized shape includes the raw body in the error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"unexpected":"shape"}`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.CreateIssueWebhook("https://example.com/webhook", "", []string{"jira:issue_created"})
		require.ErrorContains(t, err, `{"unexpected":"shape"}`)
	})

	// Regression test: Atlassian's schema requires the "jqlFilter" key to be present even when
	// empty (an empty value matches every project) - an `omitempty` tag would silently drop the
	// key for an unfiltered registration and make the whole request fail.
	t.Run("an empty jqlFilter is still sent as an explicit key", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[{"createdWebhookId":1000}]`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.CreateIssueWebhook("https://example.com/webhook", "", []string{"jira:issue_created"})
		require.NoError(t, err)

		body, _ := io.ReadAll(httpContext.Requests[0].Body)
		assert.Contains(t, string(body), `"jqlFilter":""`)
	})
}

func Test__Client__DeleteIssueWebhooks(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = client.DeleteIssueWebhooks([]int64{1000})
		require.NoError(t, err)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
		body, _ := io.ReadAll(httpContext.Requests[0].Body)
		assert.Contains(t, string(body), "1000")
	})

	t.Run("no-op for an empty id list", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = client.DeleteIssueWebhooks(nil)
		require.NoError(t, err)
		assert.Empty(t, httpContext.Requests)
	})
}

func Test__Client__RefreshIssueWebhooks(t *testing.T) {
	t.Run("successful refresh", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"expirationDate":"2026-08-27T00:00:00.000Z"}`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = client.RefreshIssueWebhooks([]int64{1000})
		require.NoError(t, err)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/webhook/refresh")
		body, _ := io.ReadAll(httpContext.Requests[0].Body)
		assert.Contains(t, string(body), "1000")
	})

	t.Run("no-op for an empty id list", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = client.RefreshIssueWebhooks(nil)
		require.NoError(t, err)
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["webhook not found"]}`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = client.RefreshIssueWebhooks([]int64{1000})
		require.ErrorContains(t, err, "webhook not found")
	})
}

func Test__Client__ExecRequestWithStatus(t *testing.T) {
	t.Run("401 self-heals via a reactive refresh and retries the request", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"refreshed-access","refresh_token":"refreshed-refresh","expires_in":3600}`,
				))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"ok":true}`))},
			},
		}

		appCtx := newAuthorizedIntegration()
		appCtx.Configuration = map[string]any{"clientId": "client-1", "clientSecret": "secret-1"}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		body, status, err := client.execRequestWithStatus(http.MethodGet, testProxyURL("/rest/api/3/issue/createmeta/TEST/issuetypes"), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		assert.Contains(t, string(body), `"ok":true`)

		require.Len(t, httpContext.Requests, 3)
		assert.Equal(t, TokenURL, httpContext.Requests[1].URL.String())

		accessToken, _ := findSecret(appCtx, SecretOAuthAccessToken)
		assert.Equal(t, "refreshed-access", accessToken)
	})

	t.Run("refresh failure on a 401 is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`))},
			},
		}

		appCtx := newAuthorizedIntegration()
		delete(appCtx.CurrentSecrets, SecretOAuthRefreshToken)
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		_, _, err = client.execRequestWithStatus(http.MethodGet, testProxyURL("/rest/api/3/issue/createmeta/TEST/issuetypes"), nil)
		require.ErrorContains(t, err, "token refresh failed")
	})

	t.Run("non-401 non-2xx status is returned without erroring", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"errorMessages":["not found"]}`))},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, status, err := client.execRequestWithStatus(http.MethodGet, testProxyURL("/rest/api/3/issue/createmeta/TEST/issuetypes"), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, status)
	})
}

func Test__Client__GetProjectIssueTypes(t *testing.T) {
	t.Run("returns issue types for a project", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"issueTypes": [
							{"id":"10001","name":"Task","subtask":false},
							{"id":"10002","name":"Bug","subtask":false},
							{"id":"10003","name":"Subtask","subtask":true}
						]
					}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		types, err := client.GetProjectIssueTypes("TEST")
		require.NoError(t, err)
		require.Len(t, types, 3)
		assert.Equal(t, "Task", types[0].Name)
		assert.Equal(t, "Bug", types[1].Name)
		assert.True(t, types[2].Subtask)
		assert.Contains(t, httpContext.Requests[0].URL.String(), testProxyURL("/rest/api/3/issue/createmeta/TEST/issuetypes"))
	})

	t.Run("project not found -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Project not found"]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.GetProjectIssueTypes("MISSING")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}

func Test__Client__GetWorkflowSchemeForProject(t *testing.T) {
	t.Run("custom scheme with id resolves full details", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"values": [
							{"projectIds":["10000"],"workflowScheme":{"id":"42","name":"Custom Scheme","defaultWorkflow":"Custom WF"}}
						]
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"id":"42","name":"Custom Scheme","defaultWorkflow":"Custom WF",
						"issueTypeMappings":{"10001":"Bug WF"}
					}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		scheme, err := client.GetWorkflowSchemeForProject("10000")
		require.NoError(t, err)
		require.NotNil(t, scheme)
		assert.Equal(t, "Custom Scheme", scheme.Name)
		assert.Equal(t, "Bug WF", scheme.IssueTypeMappings["10001"])
		// Both endpoints were hit: project assignment, then full scheme by id.
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/rest/api/3/workflowscheme/42")
	})

	t.Run("default scheme without id falls back to inlined default workflow", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"values": [
							{"projectIds":["10000"],"workflowScheme":{"name":"Default Workflow Scheme","defaultWorkflow":"jira","issueTypeMappings":{"10001":"Bug WF"}}}
						]
					}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		scheme, err := client.GetWorkflowSchemeForProject("10000")
		require.NoError(t, err)
		require.NotNil(t, scheme)
		assert.Equal(t, "Default Workflow Scheme", scheme.Name)
		assert.Equal(t, "jira", scheme.DefaultWorkflow)
		// Per-issue-type mappings inlined in the project response are preserved,
		// not discarded in favour of the default workflow.
		assert.Equal(t, "Bug WF", scheme.IssueTypeMappings["10001"])
		// No id means no second request to resolve full details.
		assert.Len(t, httpContext.Requests, 1)
	})

	t.Run("team-managed project with empty list returns nil", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		scheme, err := client.GetWorkflowSchemeForProject("10000")
		require.NoError(t, err)
		assert.Nil(t, scheme)
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
		assert.Equal(t, "Hello world", result.Content[0].Content[0].Text)
	})

	t.Run("empty string returns nil", func(t *testing.T) {
		result := WrapInADF("")
		assert.Nil(t, result)
	})
}

func Test__Client__ListServiceDesksAndRequestTypes(t *testing.T) {
	t.Run("list service desks", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"1","projectName":"SD","projectKey":"SD"}],"isLastPage":true}`)),
				},
			},
		}
		appCtx := &contexts.IntegrationContext{
			Metadata:       Metadata{CloudID: testCloudID},
			CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		desks, err := client.ListServiceDesks()
		require.NoError(t, err)
		require.Len(t, desks, 1)
		assert.Equal(t, "1", desks[0].ID)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/servicedeskapi/servicedesk?")
	})

	t.Run("list request types for desk", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"10","name":"Incident","practice":"ITSM_INCIDENT"}],"isLastPage":true}`)),
				},
			},
		}
		appCtx := &contexts.IntegrationContext{
			Metadata:       Metadata{CloudID: testCloudID},
			CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		rts, err := client.ListRequestTypes("1")
		require.NoError(t, err)
		require.Len(t, rts, 1)
		assert.Equal(t, "10", rts[0].ID)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/servicedeskapi/servicedesk/1/requesttype")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "expand=practice")
	})

	t.Run("list service desks paginates by returned count", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"1","projectName":"A","projectKey":"A"}],"isLastPage":false}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"2","projectName":"B","projectKey":"B"}],"isLastPage":true}`)),
				},
			},
		}
		appCtx := &contexts.IntegrationContext{
			Metadata:       Metadata{CloudID: testCloudID},
			CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		desks, err := client.ListServiceDesks()
		require.NoError(t, err)
		require.Len(t, desks, 2)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "start=1")
	})
}

func Test__jqlQuotedProjectKey(t *testing.T) {
	assert.Equal(t, `IT`, jqlQuotedProjectKey("IT"))
	assert.Equal(t, `IT\"X`, jqlQuotedProjectKey(`IT"X`))
	assert.Equal(t, `IT\\`, jqlQuotedProjectKey(`IT\`))
	assert.Equal(t, `a\\b\"c`, jqlQuotedProjectKey(`a\b"c`))
}

func Test__IsIncidentManagementRequestPractice(t *testing.T) {
	t.Parallel()
	assert.True(t, IsIncidentManagementRequestPractice("ITSM_INCIDENT"))
	assert.True(t, IsIncidentManagementRequestPractice("INCIDENT_MANAGEMENT"))
	assert.True(t, IsIncidentManagementRequestPractice("Incident management"))
	assert.True(t, IsIncidentManagementRequestPractice("  incident_management  "))
	assert.False(t, IsIncidentManagementRequestPractice(""))
	assert.False(t, IsIncidentManagementRequestPractice("SERVICE_REQUEST"))
	assert.False(t, IsIncidentManagementRequestPractice("POST_INCIDENT_REVIEW"))
	assert.False(t, IsIncidentManagementRequestPractice("Post-incident review"))
	assert.False(t, IsIncidentManagementRequestPractice("ITSM_POST_INCIDENT"))
}

func Test__Client__IncidentsAPI(t *testing.T) {
	cloudID := "35273b54-3f06-40d2-880f-dd28cf6daafa"

	t.Run("create incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10050","key":"ITSM-30","self":"https://test.atlassian.net/rest/api/3/issue/10050"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Metadata:       Metadata{CloudID: testCloudID},
			CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		resp, err := client.CreateIncident(cloudID, &CreateIncidentAPIRequest{
			ServiceDeskID: "6",
			RequestTypeID: "75",
			Fields:        map[string]any{"summary": "Outage"},
		})
		require.NoError(t, err)
		assert.Equal(t, "ITSM-30", resp.Key)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "api.atlassian.com/jsm/incidents/cloudId/"+cloudID+"/v1/incident")
	})

	t.Run("get incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"summary":"Sev1","priority":{"name":"High","id":"1"}}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Metadata:       Metadata{CloudID: testCloudID},
			CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		dto, err := client.GetIncident(cloudID, "10050")
		require.NoError(t, err)
		assert.Equal(t, "Sev1", dto["summary"])
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/v1/incident/10050")
	})

	t.Run("delete incident", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Metadata:       Metadata{CloudID: testCloudID},
			CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		err = client.DeleteIncident(cloudID, "10050")
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})
}

func Test__Client__HeartbeatsAPI(t *testing.T) {
	cloudID := "35273b54-3f06-40d2-880f-dd28cf6daafa"
	teamID := "4b26961a-a837-49d2-a1fe-0973013e3c3b"

	appCtx := &contexts.IntegrationContext{
		Metadata:       Metadata{CloudID: testCloudID},
		CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
	}

	t.Run("list ops teams array response", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"teamId":"` + teamID + `","teamName":"On-call"}]`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		teams, err := client.ListOpsTeams(cloudID)
		require.NoError(t, err)
		require.Len(t, teams, 1)
		assert.Equal(t, "On-call", teams[0].TeamName)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/jsm/ops/api/"+cloudID+"/v1/teams")
	})

	t.Run("list ops teams wrapped response", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"platformTeams":[{"teamId":"` + teamID + `","teamName":"On-call"}]}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		teams, err := client.ListOpsTeams(cloudID)
		require.NoError(t, err)
		require.Len(t, teams, 1)
		assert.Equal(t, teamID, teams[0].TeamID)
	})

	t.Run("create heartbeat", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"name":"DNS Checker","interval":5,"intervalUnit":"minutes","enabled":true}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		enabled := true
		resp, err := client.CreateHeartbeat(cloudID, teamID, &CreateHeartbeatRequest{
			Name:         "DNS Checker",
			Interval:     5,
			IntervalUnit: "minutes",
			Enabled:      &enabled,
		})
		require.NoError(t, err)
		assert.Equal(t, "DNS Checker", resp.Name)
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/teams/"+teamID+"/heartbeats")
	})

	t.Run("ping heartbeat", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader(`{"message":"PONG - Heartbeat received"}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		resp, err := client.PingHeartbeat(cloudID, teamID, "DNS Checker")
		require.NoError(t, err)
		assert.Equal(t, "PONG - Heartbeat received", resp.Message)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/heartbeats/ping")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "name=DNS+Checker")
	})

	t.Run("update heartbeat", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"name":"DNS Checker","interval":10,"intervalUnit":"minutes","enabled":false}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		interval := 10
		enabled := false
		resp, err := client.UpdateHeartbeat(cloudID, teamID, "DNS Checker", &UpdateHeartbeatRequest{
			Interval: &interval,
			Enabled:  &enabled,
		})
		require.NoError(t, err)
		assert.Equal(t, 10, resp.Interval)
		assert.Equal(t, http.MethodPatch, httpContext.Requests[0].Method)
	})

	t.Run("delete heartbeat", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		err = client.DeleteHeartbeat(cloudID, teamID, "DNS Checker")
		require.NoError(t, err)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})

	t.Run("list heartbeats follows links.next pagination", func(t *testing.T) {
		page2URL := fmt.Sprintf("/jsm/ops/api/%s/v1/teams/%s/heartbeats?offset=1&size=1", cloudID, teamID)
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"values":[{"name":"HB-1"}],"links":{"next":"` + page2URL + `"}}`,
					)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"values":[{"name":"HB-2"}]}`,
					)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		heartbeats, err := client.ListHeartbeats(cloudID, teamID)
		require.NoError(t, err)
		require.Len(t, heartbeats, 2)
		assert.Equal(t, "HB-1", heartbeats[0].Name)
		assert.Equal(t, "HB-2", heartbeats[1].Name)
		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[1].URL.String(), "offset=1")
	})

	t.Run("list heartbeats single page", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"values":[{"name":"HB-1"},{"name":"HB-2"}]}`,
					)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		heartbeats, err := client.ListHeartbeats(cloudID, teamID)
		require.NoError(t, err)
		require.Len(t, heartbeats, 2)
		require.Len(t, httpContext.Requests, 1)
	})
}

func Test__parseOpsTeamsResponse(t *testing.T) {
	t.Run("array", func(t *testing.T) {
		teams, err := parseOpsTeamsResponse([]byte(`[{"teamId":"abc","teamName":"Ops"}]`))
		require.NoError(t, err)
		require.Len(t, teams, 1)
		assert.Equal(t, "abc", teams[0].TeamID)
		assert.Equal(t, "Ops", teams[0].TeamName)
	})

	t.Run("wrapped platformTeams", func(t *testing.T) {
		teams, err := parseOpsTeamsResponse([]byte(`{"platformTeams":[{"teamId":"abc","teamName":"Ops"}]}`))
		require.NoError(t, err)
		require.Len(t, teams, 1)
	})

	t.Run("empty", func(t *testing.T) {
		teams, err := parseOpsTeamsResponse([]byte(`[]`))
		require.NoError(t, err)
		assert.Empty(t, teams)
	})
}

func Test__Client__ResolveNumericIssueID(t *testing.T) {
	t.Run("numeric passthrough", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{}}
		appCtx := &contexts.IntegrationContext{
			Metadata:       Metadata{CloudID: testCloudID},
			CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		id, err := client.ResolveNumericIssueID(" 10050 ")
		require.NoError(t, err)
		assert.Equal(t, "10050", id)
		assert.Len(t, httpContext.Requests, 0)
	})

	t.Run("resolve by key", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"999","key":"ITSM-1","self":"x","fields":{}}`)),
				},
			},
		}
		appCtx := &contexts.IntegrationContext{
			Metadata:       Metadata{CloudID: testCloudID},
			CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		id, err := client.ResolveNumericIssueID("ITSM-1")
		require.NoError(t, err)
		assert.Equal(t, "999", id)
	})
}

func Test__GetWorkflowStatusesByName(t *testing.T) {
	t.Run("returns statuses for an exact-name match", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{"values":[{"id":{"name":"task-workflow"},"statuses":[
						{"id":"10001","name":"To Do","statusCategory":"TODO"},
						{"id":"10002","name":"In Progress","statusCategory":"IN_PROGRESS"},
						{"id":"10003","name":"Done","statusCategory":"DONE"}
					]}]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		statuses, err := client.GetWorkflowStatusesByName("task-workflow")
		require.NoError(t, err)
		require.Len(t, statuses, 3)
		assert.Equal(t, Status{ID: "10001", Name: "To Do", Category: "TODO"}, statuses[0])
		assert.Equal(t, Status{ID: "10002", Name: "In Progress", Category: "IN_PROGRESS"}, statuses[1])
		assert.Equal(t, Status{ID: "10003", Name: "Done", Category: "DONE"}, statuses[2])
	})

	t.Run("filters out workflows whose name does not match exactly", func(t *testing.T) {
		// Jira's workflow/search does a prefix match, so a query for
		// "task" can return "task-workflow-old" too. We must not return
		// that one's statuses as if they belonged to the requested
		// workflow.
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{"values":[{"id":{"name":"task-workflow-old"},"statuses":[
						{"id":"99","name":"Stale","statusCategory":"TODO"}
					]}]}`)),
				},
			},
		}

		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.GetWorkflowStatusesByName("task-workflow")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `workflow "task-workflow" not found`)
	})
}

func Test__Client__OpsAlertsAPI(t *testing.T) {
	cloudID := "35273b54-3f06-40d2-880f-dd28cf6daafa"
	appCtx := &contexts.IntegrationContext{
		Metadata:       Metadata{CloudID: testCloudID},
		CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
	}

	t.Run("create alert", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"result":"Request will be processed","requestId":"r1","took":0.1}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		out, err := client.CreateOpsAlert(cloudID, &OpsCreateAlertRequest{Message: "Hi"})
		require.NoError(t, err)
		assert.Equal(t, "r1", out.RequestID)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "api.atlassian.com/jsm/ops/api/"+cloudID+"/v1/alerts")
	})

	t.Run("get alert", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"a1","message":"m"}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		m, err := client.GetOpsAlert(cloudID, "a1")
		require.NoError(t, err)
		assert.Equal(t, "m", m["message"])
	})

	t.Run("assign alert", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader(`{"requestId":"as1"}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		out, err := client.AssignOpsAlert(cloudID, "a1", "bb4d9938-c3c2-455d-aaab-727aa701c0d8")
		require.NoError(t, err)
		assert.Equal(t, "as1", out.RequestID)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/assign")
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	})

	t.Run("resolve request ignores premature alertId until processed", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"alertId":"stale-id","isSuccess":true}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"processedAt":"2026-05-01T00:00:00Z","alertId":"new-id","isSuccess":true,"status":"Created"}`,
					)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		id, err := client.ResolveAlertIDAfterOpsRequest(cloudID, "req-1", "")
		require.NoError(t, err)
		assert.Equal(t, "new-id", id)
		require.Len(t, httpContext.Requests, 2)
	})

	t.Run("resolve request fails when processed but not successful", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"processedAt":"2026-05-01T00:00:00Z","isSuccess":false,"status":"Invalid priority"}`,
					)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		_, err = client.ResolveAlertIDAfterOpsRequest(cloudID, "req-2", "")
		require.ErrorContains(t, err, "failed")
		require.ErrorContains(t, err, "Invalid priority")
	})

	t.Run("resolve request fails on success flag with error status text", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"processedAt":"2026-05-01T00:00:00Z","isSuccess":true,"status":"Alert does not exist","alertId":"old-1"}`,
					)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		_, err = client.ResolveAlertIDAfterOpsRequest(cloudID, "req-3", "")
		require.ErrorContains(t, err, "Alert does not exist")
	})

	t.Run("delete alert", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(strings.NewReader(`{"requestId":"d1"}`)),
				},
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		out, err := client.DeleteOpsAlert(cloudID, "a1")
		require.NoError(t, err)
		assert.Equal(t, "d1", out.RequestID)
	})
}

func Test__Auth__ExchangeCode(t *testing.T) {
	t.Run("sends the authorization code grant as JSON", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`{"access_token":"at","refresh_token":"rt","expires_in":3600}`,
				))},
			},
		}

		auth := NewAuth(httpContext)
		token, err := auth.ExchangeCode("client-1", "secret-1", "the-code", "https://sp.example.com/callback")
		require.NoError(t, err)
		assert.Equal(t, "at", token.AccessToken)
		assert.Equal(t, "rt", token.RefreshToken)
		assert.Equal(t, 3600, token.ExpiresIn)

		require.Len(t, httpContext.Requests, 1)
		request := httpContext.Requests[0]
		assert.Equal(t, TokenURL, request.URL.String())
		assert.Equal(t, "application/json", request.Header.Get("Content-Type"))

		body, readErr := io.ReadAll(request.Body)
		require.NoError(t, readErr)
		assert.Contains(t, string(body), `"grant_type":"authorization_code"`)
		assert.Contains(t, string(body), `"code":"the-code"`)
	})

	t.Run("non-2xx response -> error with body", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"invalid_grant"}`))},
			},
		}

		auth := NewAuth(httpContext)
		_, err := auth.ExchangeCode("client-1", "secret-1", "bad-code", "https://sp.example.com/callback")
		require.ErrorContains(t, err, "invalid_grant")
	})

	t.Run("response without access token -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}},
		}

		auth := NewAuth(httpContext)
		_, err := auth.ExchangeCode("client-1", "secret-1", "the-code", "https://sp.example.com/callback")
		require.ErrorContains(t, err, "missing access_token")
	})
}

func Test__Auth__RefreshToken(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"access_token":"new-at","refresh_token":"new-rt","expires_in":3600}`,
			))},
		},
	}

	auth := NewAuth(httpContext)
	token, err := auth.RefreshToken("client-1", "secret-1", "old-rt")
	require.NoError(t, err)
	assert.Equal(t, "new-at", token.AccessToken)
	assert.Equal(t, "new-rt", token.RefreshToken)

	require.Len(t, httpContext.Requests, 1)
	body, readErr := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, readErr)
	assert.Contains(t, string(body), `"grant_type":"refresh_token"`)
	assert.Contains(t, string(body), `"refresh_token":"old-rt"`)
}

func Test__Auth__HandleCallback(t *testing.T) {
	auth := NewAuth(&contexts.HTTPContext{})

	t.Run("provider error is surfaced", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?error=access_denied&error_description=denied", nil)

		_, err := auth.HandleCallback(request, "client-1", "secret-1", "state", "https://cb")
		require.ErrorContains(t, err, "access_denied")
	})

	t.Run("missing code -> error", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?state=state", nil)

		_, err := auth.HandleCallback(request, "client-1", "secret-1", "state", "https://cb")
		require.ErrorContains(t, err, "missing code or state")
	})

	t.Run("state mismatch -> error", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=other", nil)

		_, err := auth.HandleCallback(request, "client-1", "secret-1", "state", "https://cb")
		require.ErrorContains(t, err, "invalid state")
	})

	// An integration whose state was never generated must reject every callback,
	// even one carrying an attacker-supplied non-empty state.
	t.Run("empty expected state never matches", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=attacker-state", nil)

		_, err := auth.HandleCallback(request, "client-1", "secret-1", "", "https://cb")
		require.ErrorContains(t, err, "invalid state")
	})

	t.Run("valid callback exchanges the code", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"access_token":"at","expires_in":3600}`))},
			},
		}

		request := httptest.NewRequest(http.MethodGet, "/callback?code=c&state=state", nil)

		token, err := NewAuth(httpContext).HandleCallback(request, "client-1", "secret-1", "state", "https://cb")
		require.NoError(t, err)
		assert.Equal(t, "at", token.AccessToken)
	})
}

func Test__Auth__AccessibleResources(t *testing.T) {
	t.Run("returns accessible sites", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
					`[{"id":"cloud-1","name":"Test Site","url":"https://test.atlassian.net","scopes":["read:jira-work"]}]`,
				))},
			},
		}

		resources, err := NewAuth(httpContext).AccessibleResources("access-1")
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "cloud-1", resources[0].ID)
		assert.Equal(t, "https://test.atlassian.net", resources[0].URL)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "Bearer access-1", httpContext.Requests[0].Header.Get("Authorization"))
	})

	t.Run("no accessible sites is an error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))}},
		}

		_, err := NewAuth(httpContext).AccessibleResources("access-1")
		require.ErrorContains(t, err, "no accessible Jira sites")
	})
}

func Test__Client__CreateAlertWebhookIntegration(t *testing.T) {
	t.Run("successful create", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"int-1","name":"SuperPlane webhook"}`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		integration, err := client.CreateAlertWebhookIntegration(testCloudID, "SuperPlane webhook", "https://sp.test/webhooks/w1", "team-1")
		require.NoError(t, err)
		assert.Equal(t, "int-1", integration.ID)

		require.Len(t, httpContext.Requests, 1)
		req := httpContext.Requests[0]
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Contains(t, req.URL.String(), "/jsm/ops/api/"+testCloudID+"/v1/integrations")
		body, _ := io.ReadAll(req.Body)
		assert.Contains(t, string(body), `"type":"Webhook"`)
		assert.Contains(t, string(body), `"url":"https://sp.test/webhooks/w1"`)
		assert.Contains(t, string(body), `"teamId":"team-1"`)
	})

	t.Run("response missing an id is an error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.CreateAlertWebhookIntegration(testCloudID, "SuperPlane webhook", "https://sp.test/webhooks/w1", "")
		require.ErrorContains(t, err, "missing id")
	})

	t.Run("failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"message":"requires a Premium or Enterprise plan"}`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		_, err = client.CreateAlertWebhookIntegration(testCloudID, "SuperPlane webhook", "https://sp.test/webhooks/w1", "")
		require.ErrorContains(t, err, "Premium or Enterprise plan")
	})
}

func Test__Client__DeleteAlertWebhookIntegration(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			},
		}
		client, err := NewClient(httpContext, newAuthorizedIntegration())
		require.NoError(t, err)

		err = client.DeleteAlertWebhookIntegration(testCloudID, "int-1")
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/jsm/ops/api/"+testCloudID+"/v1/integrations/int-1")
	})
}

func Test__TokenResponse__GetExpiration(t *testing.T) {
	t.Run("half the token lifetime", func(t *testing.T) {
		response := TokenResponse{ExpiresIn: 3600}
		assert.Equal(t, 1800, int(response.GetExpiration().Seconds()))
	})

	t.Run("defaults to 30 minutes when missing", func(t *testing.T) {
		response := TokenResponse{}
		assert.Equal(t, 1800, int(response.GetExpiration().Seconds()))
	})
}
