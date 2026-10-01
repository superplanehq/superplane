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
