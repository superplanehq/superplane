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

	t.Run("missing project -> empty list, no API call", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}
		resources, err := j.ListResources("assignee", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
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

func Test__ListResources__Unknown(t *testing.T) {
	j := &Jira{}
	appCtx := newAuthorizedIntegration()

	resources, err := j.ListResources("nope", core.ListResourcesContext{
		Integration: appCtx,
	})

	require.NoError(t, err)
	assert.Empty(t, resources)
}
