package jira

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const globalStatusesResponse = `{
	"isLast": true,
	"values": [
		{"id":"1","name":"To Do","statusCategory":"TODO"},
		{"id":"2","name":"In Progress","statusCategory":"IN_PROGRESS"},
		{"id":"3","name":"Done","statusCategory":"DONE"}
	]
}`

func Test__ListResources__Project(t *testing.T) {
	j := &Jira{}
	appCtx := newAuthorizedIntegrationWithMetadata(Metadata{
		Projects: []Project{
			{ID: "10000", Key: "TEST", Name: "Test Project"},
			{ID: "10001", Key: "DEMO", Name: "Demo Project"},
		},
	})

	resources, err := j.ListResources("project", core.ListResourcesContext{
		Integration: appCtx,
	})

	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "project", resources[0].Type)
	assert.Equal(t, "TEST", resources[0].ID)
	assert.Contains(t, resources[0].Name, "Test Project")
}

func Test__ListResources__Project__FetchesLiveProjects(t *testing.T) {
	j := &Jira{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`[{"id":"10033","key":"SUP","name":"Superdent"}]`)),
			},
		},
	}

	resources, err := j.ListResources("project", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: newAuthorizedIntegrationWithMetadata(Metadata{}),
	})

	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "SUP", resources[0].ID)
	assert.Equal(t, "Superdent (SUP)", resources[0].Name)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/project")
}

func Test__ListResources__IssueType(t *testing.T) {
	j := &Jira{}

	t.Run("returns issue types for the project parameter", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"issueTypes": [
							{"id":"10001","name":"Task"},
							{"id":"10002","name":"Bug"}
						]
					}`)),
				},
			},
		}

		appCtx := newAuthorizedIntegration()
		resources, err := j.ListResources("issueType", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Parameters:  map[string]string{"project": "TEST"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "issueType", resources[0].Type)
		assert.Equal(t, "Task", resources[0].Name)
		assert.Equal(t, "Task", resources[0].ID)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/issue/createmeta/TEST/issuetypes")
	})

	t.Run("missing project parameter -> empty list (no API call)", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		appCtx := newAuthorizedIntegration()

		resources, err := j.ListResources("issueType", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: appCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("unresolved expression project parameter -> empty list", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		appCtx := newAuthorizedIntegration()

		resources, err := j.ListResources("issueType", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: appCtx,
			Parameters:  map[string]string{"project": "{{ trigger.project }}"},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
		assert.Empty(t, httpContext.Requests)
	})
}

func Test__ListResources__Assignee(t *testing.T) {
	j := &Jira{}

	t.Run("returns assignable users for the project", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"accountId":"acct-1","displayName":"Alice","emailAddress":"alice@example.com"},
						{"accountId":"acct-2","displayName":"Bob"}
					]`)),
				},
			},
		}

		resources, err := j.ListResources("assignee", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Parameters:  map[string]string{"project": "TEST"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "acct-1", resources[0].ID)
		assert.Contains(t, resources[0].Name, "Alice")
		assert.Contains(t, resources[0].Name, "alice@example.com")
		assert.Equal(t, "Bob", resources[1].Name)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/user/assignable/search")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "project=TEST")
	})

	t.Run("missing project and metadata -> empty list, no API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		resources, err := j.ListResources("assignee", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
		})
		require.NoError(t, err)
		assert.Empty(t, resources)
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("missing project uses first synced project from metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"accountId":"acct-1","displayName":"Alice"}
					]`)),
				},
			},
		}

		resources, err := j.ListResources("assignee", core.ListResourcesContext{
			HTTP: httpContext,
			Integration: newAuthorizedIntegrationWithMetadata(Metadata{
				Projects: []Project{{Key: "SYNC", Name: "Synced"}},
			}),
		})
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "acct-1", resources[0].ID)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "project=SYNC")
	})
}

func Test__ListResources__JSMApproval(t *testing.T) {
	j := &Jira{}

	t.Run("returns only pending approvals for the issueKey parameter", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"values": [
							{"id":"1","name":"Manager","finalDecision":"approved"},
							{"id":"2","name":"Director","finalDecision":"PENDING"},
							{"id":"3","finalDecision":"PENDING"}
						],
						"isLastPage": true
					}`)),
				},
			},
		}

		resources, err := j.ListResources("jsmApproval", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Parameters:  map[string]string{"issueKey": "ITSM-1"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "jsmApproval", resources[0].Type)
		assert.Equal(t, "2", resources[0].ID)
		assert.Equal(t, "Director", resources[0].Name)
		assert.Equal(t, "3", resources[1].ID)
		assert.Equal(t, "Approval 3", resources[1].Name)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/servicedeskapi/request/ITSM-1/approval")
	})

	t.Run("missing issueKey -> empty list, no API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		resources, err := j.ListResources("jsmApproval", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("unresolved expression issueKey -> empty list, no API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		resources, err := j.ListResources("jsmApproval", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Parameters:  map[string]string{"issueKey": "{{ trigger.issueKey }}"},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
		assert.Empty(t, httpContext.Requests)
	})
}

func Test__ListResources__Priority(t *testing.T) {
	j := &Jira{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"id":"1","name":"Highest"},
					{"id":"3","name":"Medium"},
					{"id":"5","name":"Lowest"}
				]`)),
			},
		},
	}

	resources, err := j.ListResources("priority", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: newAuthorizedIntegration(),
	})

	require.NoError(t, err)
	require.Len(t, resources, 3)
	assert.Equal(t, "Highest", resources[0].Name)
	assert.Equal(t, "Highest", resources[0].ID)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/priority")
}

func Test__ListResources__Priority__MissingHTTPContext(t *testing.T) {
	j := &Jira{}

	resources, err := j.ListResources("priority", core.ListResourcesContext{
		Integration: newAuthorizedIntegration(),
	})

	require.NoError(t, err)
	assert.Empty(t, resources)
}

func Test__ListResources__IssueStatus(t *testing.T) {
	j := &Jira{}

	t.Run("with project parameter -> uses project statuses endpoint", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"name":"Task","statuses":[
							{"id":"1","name":"To Do","statusCategory":{"key":"new"}},
							{"id":"2","name":"Done","statusCategory":{"key":"done"}}
						]}
					]`)),
				},
			},
		}

		resources, err := j.ListResources("issueStatus", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Parameters:  map[string]string{"project": "TEST"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "issueStatus", resources[0].Type)
		assert.Equal(t, "To Do", resources[0].Name)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/project/TEST/statuses")
	})

	t.Run("without project parameter -> falls back to global statuses", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(globalStatusesResponse)),
				},
			},
		}

		resources, err := j.ListResources("issueStatus", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
		})

		require.NoError(t, err)
		require.Len(t, resources, 3)
		assert.Equal(t, "To Do", resources[0].Name)
		assert.Equal(t, "In Progress", resources[1].Name)
		assert.Equal(t, "Done", resources[2].Name)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/statuses/search")
	})

	t.Run("unresolved expression project parameter -> falls back to global statuses", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(globalStatusesResponse)),
				},
			},
		}

		resources, err := j.ListResources("issueStatus", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Parameters:  map[string]string{"project": "{{ trigger.project }}"},
		})

		require.NoError(t, err)
		require.Len(t, resources, 3)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/statuses/search")
	})
}

func Test__ListResources__Unknown(t *testing.T) {
	j := &Jira{}
	appCtx := newAuthorizedIntegration()

	resources, err := j.ListResources("nope", core.ListResourcesContext{
		Integration: appCtx,
	})

	require.NoError(t, err)
	assert.Empty(t, resources)
}

func Test__Jira__ListResources__serviceDesk(t *testing.T) {
	j := &Jira{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"1","projectName":"Help","projectKey":"HEL"}],"isLastPage":true}`)),
			},
		},
	}
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl":  "https://test.atlassian.net",
			"email":    "a@b.com",
			"apiToken": "token",
		},
	}

	resources, err := j.ListResources("serviceDesk", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: appCtx,
		Parameters:  map[string]string{"type": "serviceDesk"},
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "1", resources[0].ID)
	assert.Contains(t, resources[0].Name, "Help")
}

func Test__Jira__ListResources__serviceDeskRequestType(t *testing.T) {
	j := &Jira{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"99","name":"Get help","practice":"ITSM_INCIDENT"}],"isLastPage":true}`)),
			},
		},
	}
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl":  "https://test.atlassian.net",
			"email":    "a@b.com",
			"apiToken": "token",
		},
	}

	resources, err := j.ListResources("serviceDeskRequestType", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: appCtx,
		Parameters:  map[string]string{"type": "serviceDeskRequestType", "serviceDesk": "1"},
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "99", resources[0].ID)
}

func Test__Jira__ListResources__serviceDeskRequestType_emptyDesk(t *testing.T) {
	j := &Jira{}
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "token",
		},
	}
	resources, err := j.ListResources("serviceDeskRequestType", core.ListResourcesContext{
		HTTP:        &contexts.HTTPContext{},
		Integration: appCtx,
		Parameters:  map[string]string{"type": "serviceDeskRequestType"},
	})
	require.NoError(t, err)
	assert.Len(t, resources, 0)
}

func Test__Jira__ListResources__impact(t *testing.T) {
	j := &Jira{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"6","projectName":"IT","projectKey":"IT"}],"isLastPage":true}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{"requestTypeFields":[
					{"fieldId":"customfield_10020","name":"Impact","validValues":[
						{"label":"High","value":"10001"},
						{"label":"Low","value":"10002"}
					]}
				]}`)),
			},
		},
	}
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "token",
		},
	}

	resources, err := j.ListResources("impact", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: appCtx,
		Parameters: map[string]string{
			"type":                   "impact",
			"serviceDesk":            "6",
			"serviceDeskRequestType": "75",
		},
	})
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "10001", resources[0].ID)
	assert.Equal(t, "High", resources[0].Name)
}

func Test__Jira__ListResources__urgency__fieldOptionsFallback(t *testing.T) {
	j := &Jira{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"6","projectName":"IT","projectKey":"IT"}],"isLastPage":true}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{"requestTypeFields":[
					{"fieldId":"customfield_10021","name":"Urgency","validValues":[]}
				]}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"issueTypes":[{"id":"10001","name":"Incident"}]}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"fields":{"customfield_10021":{"name":"Urgency","allowedValues":[{"id":"2","name":"High"}]}}
				}`)),
			},
		},
	}
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "token",
		},
	}

	resources, err := j.ListResources("urgency", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: appCtx,
		Parameters: map[string]string{
			"type":                   "urgency",
			"serviceDesk":            "6",
			"serviceDeskRequestType": "75",
		},
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "2", resources[0].ID)
	assert.Equal(t, "High", resources[0].Name)
}

func Test__Jira__ListResources__impact_missingParams(t *testing.T) {
	j := &Jira{}
	resources, err := j.ListResources("impact", core.ListResourcesContext{
		HTTP:        &contexts.HTTPContext{},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"siteUrl": "https://x.net", "email": "a@b.com", "apiToken": "t"}},
		Parameters:  map[string]string{"type": "impact"},
	})
	require.NoError(t, err)
	assert.Len(t, resources, 0)
}

func Test__Jira__ListResources__issue(t *testing.T) {
	j := &Jira{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"1","projectName":"Help","projectKey":"HEL"}],"isLastPage":true}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{"values":[
					{"issueKey":"HEL-1","summary":"Ticket one"},
					{"issueKey":"HEL-2","summary":""}
				],"isLastPage":true}`)),
			},
		},
	}
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl":  "https://test.atlassian.net",
			"email":    "a@b.com",
			"apiToken": "token",
		},
	}

	resources, err := j.ListResources("issue", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: appCtx,
		Parameters:  map[string]string{"type": "issue", "project": "HEL"},
	})
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "HEL-1", resources[0].ID)
	assert.Contains(t, resources[0].Name, "HEL-1")
	assert.Contains(t, resources[0].Name, "Ticket one")
	assert.Equal(t, "HEL-2", resources[1].ID)
	require.Len(t, httpContext.Requests, 2)
	assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/servicedeskapi/servicedesk")
	assert.Contains(t, httpContext.Requests[1].URL.String(), "/rest/servicedeskapi/request")
}

func Test__Jira__ListResources__issue_deskEmptyFallsBackToSearch(t *testing.T) {
	j := &Jira{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"1","projectName":"Help","projectKey":"HEL"}],"isLastPage":true}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"values":[],"isLastPage":true}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"issues":[{"id":"100","key":"HEL-9","fields":{"summary":"From JQL"}}],"total":1}`)),
			},
		},
	}
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl":  "https://test.atlassian.net",
			"email":    "a@b.com",
			"apiToken": "token",
		},
	}

	resources, err := j.ListResources("issue", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: appCtx,
		Parameters:  map[string]string{"type": "issue", "project": "HEL"},
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "HEL-9", resources[0].ID)
	require.Len(t, httpContext.Requests, 3)
	assert.Equal(t, http.MethodPost, httpContext.Requests[2].Method)
	assert.True(t, strings.HasSuffix(httpContext.Requests[2].URL.Path, "/rest/api/3/search"))
}

func Test__Jira__ListResources__issue_noProject(t *testing.T) {
	j := &Jira{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"issues":[{"id":"1","key":"X-1","fields":{"summary":"S"}}]}`)),
			},
		},
	}
	appCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"siteUrl":  "https://test.atlassian.net",
			"email":    "a@b.com",
			"apiToken": "token",
		},
	}

	resources, err := j.ListResources("issue", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: appCtx,
		Parameters:  map[string]string{"type": "issue"},
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "X-1", resources[0].ID)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.True(t, strings.HasSuffix(httpContext.Requests[0].URL.Path, "/rest/api/3/search"))
}

func Test__findRequestTypeFieldID(t *testing.T) {
	fields := []RequestTypeField{
		{FieldID: "summary", Name: "Summary"},
		{FieldID: "customfield_10020", Name: "Impact"},
		{FieldID: "customfield_10021", Name: "Urgency"},
	}

	assert.Equal(t, "customfield_10020", findRequestTypeFieldID(fields, "impact"))
	assert.Equal(t, "customfield_10021", findRequestTypeFieldID(fields, "urgency"))
	assert.Equal(t, "", findRequestTypeFieldID(fields, "priority"))
}

func Test__findRequestTypeField__prefersExactWithOptions(t *testing.T) {
	fields := []RequestTypeField{
		{FieldID: "customfield_1", Name: "Business urgency rating", ValidValues: nil},
		{FieldID: "customfield_2", Name: "Urgency", ValidValues: []RequestTypeFieldValue{{Label: "High", Value: "1"}}},
	}
	f := findRequestTypeField(fields, "urgency")
	require.NotNil(t, f)
	assert.Equal(t, "customfield_2", f.FieldID)
}

func Test__requestTypeFieldResources__fromValidValues(t *testing.T) {
	field := &RequestTypeField{
		FieldID: "customfield_10020",
		Name:    "Impact",
		ValidValues: []RequestTypeFieldValue{
			{Label: "High", Value: "10001"},
			{Label: "Low", Value: "10002"},
		},
	}
	resources, err := requestTypeFieldResources(nil, field, "impact", "", "impact")
	require.NoError(t, err)
	require.Len(t, resources, 2)
	assert.Equal(t, "10001", resources[0].ID)
}

func Test__requestTypeFieldResources__createmetaFallback(t *testing.T) {
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
			"siteUrl": "https://test.atlassian.net", "email": "a@b.com", "apiToken": "token",
		},
	})
	require.NoError(t, err)

	resources, err := requestTypeFieldResources(client, nil, "urgency", "IT", "urgency")
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "5", resources[0].ID)
}

func Test__ListResources__Team(t *testing.T) {
	j := &Jira{}
	teamID := "4b26961a-a837-49d2-a1fe-0973013e3c3b"
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"platformTeams":[{"teamId":"` + teamID + `","teamName":"On-call"}]}`)),
			},
		},
	}

	resources, err := j.ListResources("team", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: jiraTestIntegration(),
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "team", resources[0].Type)
	assert.Equal(t, teamID, resources[0].ID)
	assert.Equal(t, "On-call", resources[0].Name)
}

func Test__ListResources__Heartbeat(t *testing.T) {
	j := &Jira{}
	teamID := "4b26961a-a837-49d2-a1fe-0973013e3c3b"
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"values":[{"name":"DNS Checker","status":"Responsive"}]}`)),
			},
		},
	}

	resources, err := j.ListResources("heartbeat", core.ListResourcesContext{
		HTTP:        httpContext,
		Integration: jiraTestIntegration(),
		Parameters:  map[string]string{"team": teamID},
	})
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, "DNS Checker", resources[0].ID)
	assert.Contains(t, resources[0].Name, "DNS Checker")
}

func Test__ListResources__Alert(t *testing.T) {
	j := &Jira{}
	alertID := "abc-123-def"

	t.Run("returns alerts with name and id", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{"values":[
						{"id":"` + alertID + `","message":"Disk usage high","tinyId":"7"},
						{"id":"xyz-456","message":"","tinyId":"8"}
					]}`)),
				},
			},
		}

		resources, err := j.ListResources("alert", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: jiraTestIntegration(),
		})

		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "alert", resources[0].Type)
		assert.Equal(t, alertID, resources[0].ID)
		assert.Equal(t, "Disk usage high #7", resources[0].Name)
		assert.Equal(t, "xyz-456", resources[1].ID)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/jsm/ops/api/")
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/v1/alerts")
	})

	t.Run("skips rows with empty id", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{"values":[
						{"id":"","message":"no id"},
						{"id":"good-id","message":"has id","tinyId":"1"}
					]}`)),
				},
			},
		}

		resources, err := j.ListResources("alert", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: jiraTestIntegration(),
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "good-id", resources[0].ID)
	})
}

func Test__opsAlertIntegrationResourceLabel__truncatesByRune(t *testing.T) {
	longCJK := strings.Repeat("あ", 100)
	label := opsAlertIntegrationResourceLabel(map[string]any{
		"message": longCJK,
		"tinyId":  "42",
	}, "alert-uuid")
	assert.True(t, utf8.ValidString(label))
	assert.LessOrEqual(t, utf8.RuneCountInString(label), opsAlertLabelMaxRunes+len(" #42"))
	assert.Contains(t, label, "#42")
	assert.True(t, strings.HasSuffix(label, "... #42"))

	short := opsAlertIntegrationResourceLabel(map[string]any{"message": "ok", "tinyId": "1"}, "id")
	assert.Equal(t, "ok #1", short)
}
