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

func Test__Jira__ListResources__issue(t *testing.T) {
	j := &Jira{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"issues":[{"key":"HEL-1","fields":{"summary":"Ticket one"}},{"key":"HEL-2","fields":{"summary":""}}],"isLast":true}`))},
		},
	}
	appCtx := &contexts.IntegrationContext{
		Metadata:       Metadata{CloudID: testCloudID},
		CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
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
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.True(t, strings.HasSuffix(httpContext.Requests[0].URL.Path, "/rest/api/3/search/jql"))
	body, err := io.ReadAll(httpContext.Requests[0].Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `project = \"HEL\" ORDER BY updated DESC`)
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
		Metadata:       Metadata{CloudID: testCloudID},
		CurrentSecrets: map[string]core.IntegrationSecret{SecretOAuthAccessToken: {Name: SecretOAuthAccessToken, Value: []byte(testAccessToken)}},
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
	assert.True(t, strings.HasSuffix(httpContext.Requests[0].URL.Path, "/rest/api/3/search/jql"))
}
