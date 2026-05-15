package jira

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__NewClient(t *testing.T) {
	t.Run("missing baseUrl -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, appCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "baseUrl")
	})

	t.Run("missing email -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"apiToken": "test-token",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, appCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("missing apiToken -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl": "https://test.atlassian.net",
				"email":   "test@example.com",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		_, err := NewClient(httpCtx, appCtx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "apiToken")
	})

	t.Run("successful client creation", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		httpCtx := &contexts.HTTPContext{}
		client, err := NewClient(httpCtx, appCtx)

		require.NoError(t, err)
		assert.Equal(t, "https://test.atlassian.net", client.BaseURL)
		assert.Equal(t, "test@example.com", client.Email)
		assert.Equal(t, "test-token", client.Token)
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

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		user, err := client.GetCurrentUser()

		require.NoError(t, err)
		assert.Equal(t, "123", user.AccountID)
		assert.Equal(t, "Test User", user.DisplayName)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/myself")
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

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "invalid-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
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

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
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
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[]`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
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
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10001","key":"TEST-123","self":"https://test.atlassian.net/rest/api/3/issue/10001","fields":{"summary":"Test issue"}}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
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
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Issue does not exist"]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		_, err = client.GetIssue("INVALID-999")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}

func Test__Client__SearchIssues(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{"issues":[
					{"id":"1","key":"IT-1","fields":{"summary":"First"}},
					{"id":"2","key":"IT-2","fields":{"summary":"Second"}}
				]}`)),
			},
		},
	}

	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl":  "https://test.atlassian.net",
			"email":    "test@example.com",
			"apiToken": "test-token",
		},
	}

	client, err := NewClient(httpContext, appCtx)
	require.NoError(t, err)

	hits, err := client.SearchIssues(`project = "IT" ORDER BY updated DESC`, 10)
	require.NoError(t, err)
	require.Len(t, hits, 2)
	assert.Equal(t, "IT-1", hits[0].Key)
	assert.Equal(t, "First", hits[0].Fields["summary"])
	require.Len(t, httpContext.Requests, 1)
	req := httpContext.Requests[0]
	assert.Equal(t, http.MethodPost, req.Method)
	assert.True(t, strings.HasSuffix(req.URL.Path, "/rest/api/3/search"))
	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"jql":"project = \"IT\" ORDER BY updated DESC"`)
	assert.Contains(t, string(body), `"maxResults":10`)
	assert.Contains(t, string(body), `"startAt":0`)
}

func Test__Client__SearchIssuesUpTo__paginates(t *testing.T) {
	var firstIssues []string
	for i := range 100 {
		firstIssues = append(firstIssues, fmt.Sprintf(`{"id":"%d","key":"IT-%d","fields":{}}`, i, i))
	}
	page1 := fmt.Sprintf(`{"startAt":0,"maxResults":100,"total":150,"issues":[%s]}`, strings.Join(firstIssues, ","))
	page2 := `{"startAt":100,"maxResults":100,"total":150,"issues":[{"id":"100","key":"IT-100","fields":{"summary":"Last"}}]}`

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(page1))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(page2))},
		},
	}
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl":  "https://test.atlassian.net",
			"email":    "test@example.com",
			"apiToken": "test-token",
		},
	}
	client, err := NewClient(httpContext, appCtx)
	require.NoError(t, err)

	hits, err := client.SearchIssuesUpTo(`project = "IT"`, 200)
	require.NoError(t, err)
	require.Len(t, hits, 101)
	assert.Equal(t, "IT-0", hits[0].Key)
	assert.Equal(t, "IT-100", hits[100].Key)
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
	b0, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(b0), `"startAt":0`)
	b1, err := io.ReadAll(httpContext.Requests[1].Body)
	require.NoError(t, err)
	assert.Contains(t, string(b1), `"startAt":100`)
}

func Test__Client__CreateIssue(t *testing.T) {
	t.Run("successful issue creation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10002","key":"TEST-124","self":"https://test.atlassian.net/rest/api/3/issue/10002"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
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
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(`{"id":"10003","key":"TEST-125","self":"https://test.atlassian.net/rest/api/3/issue/10003"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
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
				{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Project is required"]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)

		req := &CreateIssueRequest{
			Fields: CreateIssueFields{},
		}

		_, err = client.CreateIssue(req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
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
				"baseUrl":  "https://test.atlassian.net",
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
				"baseUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "t",
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
				"baseUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "t",
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
				"baseUrl":  "https://test.atlassian.net",
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
				"baseUrl":  "https://test.atlassian.net",
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
				"baseUrl":  "https://test.atlassian.net",
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
				"baseUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "t",
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
				"baseUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "t",
			},
		}
		client, err := NewClient(httpContext, appCtx)
		require.NoError(t, err)
		id, err := client.ResolveNumericIssueID("ITSM-1")
		require.NoError(t, err)
		assert.Equal(t, "999", id)
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
}

func Test__filterRequestTypesForIncidentsAPI(t *testing.T) {
	t.Parallel()
	all := []RequestType{
		{ID: "1", Name: "Help", Practice: "SERVICE_REQUEST"},
		{ID: "2", Name: "Outage", Practice: "ITSM_INCIDENT"},
	}
	out := filterRequestTypesForIncidentsAPI(all)
	assert.Len(t, out, 1)
	assert.Equal(t, "2", out[0].ID)
}

func Test__filterRequestTypesForIncidentsAPI_noPractice(t *testing.T) {
	t.Parallel()
	all := []RequestType{
		{ID: "1", Name: "A"},
		{ID: "2", Name: "B"},
	}
	out := filterRequestTypesForIncidentsAPI(all)
	assert.Equal(t, all, out)
}

func Test__ListCustomFieldOptions__ignoresContextAPI403(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"issueTypes":[{"id":"10001","name":"Incident"}]
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"fields":{
						"customfield_10021":{
							"name":"Urgency",
							"allowedValues":[{"id":"5","name":"High"}]
						}
					}
				}`)),
			},
		},
	}
	client, err := NewClient(httpContext, &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "token",
		},
	})
	require.NoError(t, err)

	opts := client.ListCustomFieldOptions("customfield_10021", "IT", "urgency")
	require.Len(t, opts, 1)
	assert.Equal(t, "5", opts[0].Value)
	assert.Equal(t, "High", opts[0].Label)
}
