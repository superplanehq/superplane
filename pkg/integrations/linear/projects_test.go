package linear

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Client__ListWorkspaceProjects(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"projects":{"nodes":[{"id":"p1","name":"Checkout","teams":{"nodes":[{"id":"team-b"},{"id":"team-a"}]}}],"pageInfo":{"hasNextPage":false}}}}`),
		},
	}

	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	projects, err := client.ListWorkspaceProjects()
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "p1", projects[0].ID)
	assert.Equal(t, "Checkout", projects[0].Name)
	assert.Equal(t, []string{"team-a", "team-b"}, projects[0].TeamIDs)
}

func Test__Client__ProjectTeamIDs(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"projects":{"nodes":[{"id":"p1","name":"Checkout","teams":{"nodes":[{"id":"team-b"}]}},{"id":"p2","name":"Billing","teams":{"nodes":[{"id":"team-a"},{"id":"team-b"}]}}],"pageInfo":{"hasNextPage":false}}}}`),
		},
	}

	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	teamIDs, err := client.ProjectTeamIDs([]string{" p2 ", "p1", "p1"})
	require.NoError(t, err)
	assert.Equal(t, []string{"team-a", "team-b"}, teamIDs)
	assert.Equal(t, []any{"p2", "p1"}, variablesFromRequest(t, httpContext, 0)["ids"])
}

func Test__Client__ProjectTeamIDs__UnreadableProject(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"projects":{"nodes":[{"id":"p1","name":"Checkout","teams":{"nodes":[{"id":"team-a"}]}}],"pageInfo":{"hasNextPage":false}}}}`),
		},
	}

	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	_, err = client.ProjectTeamIDs([]string{"p1", "p2"})
	require.ErrorContains(t, err, "selected projects were not found")
}

func Test__Client__ProjectTeamIDs__Missing(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(`{"data":{"projects":{"nodes":[],"pageInfo":{"hasNextPage":false}}}}`),
		},
	}

	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)

	_, err = client.ProjectTeamIDs([]string{"missing"})
	require.ErrorContains(t, err, "selected projects were not found")
}

func Test__Client__SearchOpenProjectIssues__Identifiers(t *testing.T) {
	for _, query := range []string{"ENG-123", "eng-123", "ENG 123", "please find eng-123 for me", "https://linear.app/acme/issue/eng-123/fix"} {
		t.Run(query, func(t *testing.T) {
			httpContext := &contexts.HTTPContext{Responses: []*http.Response{
				jsonResponse(`{"data":{"issues":{"nodes":[{"id":"fuzzy","identifier":"ENG-9"},{"id":"exact","identifier":"ENG-123"}]}}}`),
				jsonResponse(`{"data":{"issues":{"nodes":[{"id":"exact","identifier":"ENG-123","project":{"id":"p1"},"state":{"type":"started"}}]}}}`),
			}}
			client, err := NewClient(httpContext, newAuthorizedIntegration())
			require.NoError(t, err)
			issues, err := client.SearchOpenProjectIssues([]string{"p1"}, query, 2)
			require.NoError(t, err)
			require.Len(t, issues, 2)
			assert.Equal(t, "exact", issues[0].ID)
			assert.Equal(t, "fuzzy", issues[1].ID)
			assert.Equal(t, []any{"ENG-123"}, variablesFromRequest(t, httpContext, 1)["identifiers"])
		})
	}
}

func Test__Client__SearchProjectIssues__BareNumber(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"data":{"issues":{"nodes":[{"id":"text","identifier":"ENG-9"}]}}}`),
		jsonResponse(`{"data":{"projects":{"nodes":[{"id":"p1","teams":{"nodes":[{"id":"t1","key":"eng"},{"id":"t2","key":"OPS"}]}},{"id":"p2","teams":{"nodes":[{"id":"t1","key":"ENG"}]}}],"pageInfo":{"hasNextPage":false}}}}`),
		jsonResponse(`{"data":{"issues":{"nodes":[{"id":"eng","identifier":"ENG-123"},{"id":"ops","identifier":"OPS-123"}]}}}`),
	}}
	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)
	issues, err := client.SearchProjectIssues([]string{"p1", "p2"}, "123", 3)
	require.NoError(t, err)
	require.Len(t, issues, 3)
	assert.Equal(t, "ENG-123", issues[0].Identifier)
	assert.True(t, issues[0].ExactMatch)
	assert.Equal(t, "OPS-123", issues[1].Identifier)
	assert.True(t, issues[1].ExactMatch)
	assert.False(t, issues[2].ExactMatch)
	assert.Equal(t, []any{"ENG-123", "OPS-123"}, variablesFromRequest(t, httpContext, 2)["identifiers"])
	assert.Equal(t, []any{"p1", "p2"}, variablesFromRequest(t, httpContext, 1)["ids"])
}

func Test__Client__SearchProjectIssues__PartialIdentifiers(t *testing.T) {
	for _, query := range []string{"ENG-", "ENG-12"} {
		t.Run(query, func(t *testing.T) {
			responses := []*http.Response{jsonResponse(`{"data":{"issues":{"nodes":[{"id":"text","identifier":"OPS-7"}]}}}`)}
			if query == "ENG-12" {
				// ENG-12 means issue 12, never a numeric prefix of ENG-123.
				responses = append(responses, jsonResponse(`{"data":{"issues":{"nodes":[]}}}`))
			}
			httpContext := &contexts.HTTPContext{Responses: responses}
			client, err := NewClient(httpContext, newAuthorizedIntegration())
			require.NoError(t, err)
			issues, err := client.SearchProjectIssues([]string{"p1"}, query, 5)
			require.NoError(t, err)
			require.Len(t, issues, 1)
			assert.Equal(t, "text", issues[0].ID)
			assert.False(t, issues[0].ExactMatch)
			if query == "ENG-12" {
				assert.Equal(t, []any{"ENG-12"}, variablesFromRequest(t, httpContext, 1)["identifiers"])
			}
		})
	}
}

func Test__Client__SearchProjectIssues__ScopeAndLimit(t *testing.T) {
	for _, scope := range []string{
		`"project":{"id":"other"}`,
		`"project":null`,
		`"project":{"id":"p1"},"state":{"type":"completed"}`,
		`"project":{"id":"p1"},"state":{"type":"canceled"}`,
		`"project":{"id":"p1"},"labels":{"nodes":[]}`,
	} {
		t.Run(scope, func(t *testing.T) {
			httpContext := &contexts.HTTPContext{Responses: []*http.Response{
				jsonResponse(`{"data":{"issues":{"nodes":[{"id":"text"}]}}}`),
				jsonResponse(`{"data":{"issues":{"nodes":[{"id":"exact","identifier":"ENG-123",` + scope + `}]}}}`),
			}}
			client, err := NewClient(httpContext, newAuthorizedIntegration())
			require.NoError(t, err)
			issues, err := client.SearchProjectIssues([]string{"p1"}, "eng-123", 1)
			require.NoError(t, err)
			require.Len(t, issues, 1)
			assert.Equal(t, "exact", issues[0].ID)
			assert.True(t, issues[0].ExactMatch)
		})
	}
}

func Test__Client__SearchProjectIssues__LookupFailure(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		jsonResponse(`{"data":{"issues":{"nodes":[]}}}`),
		jsonResponse(`{"errors":[{"message":"Rate limit exceeded"}]}`),
	}}
	client, err := NewClient(httpContext, newAuthorizedIntegration())
	require.NoError(t, err)
	_, err = client.SearchProjectIssues([]string{"p1"}, "ENG-123", 5)
	require.ErrorContains(t, err, "Rate limit exceeded")
}
