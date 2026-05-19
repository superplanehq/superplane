package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const (
	testSiteURL  = "https://your-domain.atlassian.net"
	testEmail    = "user@example.com"
	testAPIToken = "test-api-token"
)

// newAuthorizedIntegration returns an IntegrationContext that has Basic Auth
// credentials and integration metadata already populated, simulating a
// successfully-configured integration.
func newAuthorizedIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl":  testSiteURL,
			"email":    testEmail,
			"apiToken": testAPIToken,
		},
		Metadata: Metadata{CloudID: "cloud-123"},
	}
}

func newAuthorizedIntegrationWithMetadata(metadata Metadata) *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl":  testSiteURL,
			"email":    testEmail,
			"apiToken": testAPIToken,
		},
		Metadata: metadata,
	}
}

func Test__NewClient(t *testing.T) {
	t.Run("missing site URL -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"email":    testEmail,
				"apiToken": testAPIToken,
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "site URL")
	})

	t.Run("missing email -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"siteUrl":  testSiteURL,
				"apiToken": testAPIToken,
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("missing API token -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"siteUrl": testSiteURL,
				"email":   testEmail,
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API token")
	})

	t.Run("empty site URL -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"siteUrl":  "",
				"email":    testEmail,
				"apiToken": testAPIToken,
			},
		}

		_, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing Jira site URL")
	})

	t.Run("successful client creation", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, newAuthorizedIntegration())

		require.NoError(t, err)
		assert.Equal(t, testSiteURL, client.SiteURL)
		assert.Equal(t, testEmail, client.Email)
		assert.Equal(t, testAPIToken, client.Token)
	})

	t.Run("trailing slash on site URL is trimmed", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"siteUrl":  testSiteURL + "/",
				"email":    testEmail,
				"apiToken": testAPIToken,
			},
		}

		client, err := NewClient(&contexts.HTTPContext{}, appCtx)
		require.NoError(t, err)
		assert.Equal(t, testSiteURL, client.SiteURL)
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testSiteURL+"/rest/api/3/myself")
		assert.True(t, strings.HasPrefix(httpContext.Requests[0].Header.Get("Authorization"), "Basic "))
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testSiteURL+"/rest/api/3/project")
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testSiteURL+"/rest/api/3/issue/TEST-123")
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testSiteURL+"/rest/api/3/issue")
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testSiteURL+"/rest/api/3/issue/TEST-1")
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
		assert.Contains(t, httpContext.Requests[0].URL.String(), testSiteURL+"/rest/api/3/issue/createmeta/TEST/issuetypes")
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

func Test__Client__FetchCloudID(t *testing.T) {
	t.Run("successful tenant_info", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"cloudId":"abc-123"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"siteUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		id, err := client.FetchCloudID()
		require.NoError(t, err)
		assert.Equal(t, "abc-123", id)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/_edge/tenant_info")
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
			Configuration: map[string]any{
				"siteUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "t",
			},
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
			Configuration: map[string]any{
				"siteUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "t",
			},
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
			Configuration: map[string]any{
				"siteUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "t",
			},
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
			Configuration: map[string]any{
				"siteUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
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
			Configuration: map[string]any{
				"siteUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
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
			Configuration: map[string]any{
				"siteUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		err = client.DeleteIncident(cloudID, "10050")
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})
}

func Test__Client__ResolveNumericIssueID(t *testing.T) {
	t.Run("numeric passthrough", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{}}
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"siteUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "t",
			},
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
			Configuration: map[string]any{
				"siteUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "t",
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		id, err := client.ResolveNumericIssueID("ITSM-1")
		require.NoError(t, err)
		assert.Equal(t, "999", id)
	})
}

func Test__NewClient__OAuth(t *testing.T) {
	t.Run("missing access token -> error", func(t *testing.T) {
		_, err := NewClient(&contexts.HTTPContext{}, &contexts.IntegrationContext{
			Metadata: Metadata{AuthType: AuthTypeOAuth, CloudID: "cloud-123"},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "OAuth accessToken not found")
	})

	t.Run("missing cloud ID -> error", func(t *testing.T) {
		_, err := NewClient(&contexts.HTTPContext{}, &contexts.IntegrationContext{
			Metadata: Metadata{AuthType: AuthTypeOAuth},
			CurrentSecrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("access-token")},
			},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cloud ID is missing")
	})

	t.Run("successful oauth client creation", func(t *testing.T) {
		client, err := NewClient(&contexts.HTTPContext{}, oauthIntegrationContext())

		require.NoError(t, err)
		assert.Equal(t, "https://api.atlassian.com/ex/jira/cloud-123", client.BaseURL)
		assert.Equal(t, AuthTypeOAuth, client.AuthType)
		assert.Equal(t, "access-token", client.Token)
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
