package bitbucket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectorySkipsAWorkspaceTheAccountIsNotIn(t *testing.T) {
	listed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repositories/acme" {
			listed = true
		}
		assert.Equal(t, "Bearer system-token", r.Header.Get("Authorization"))
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	directory := Directory{BaseURL: server.URL, HTTP: server.Client()}
	repositories, member, err := directory.VisibleRepositories(
		context.Background(),
		"system-token",
		"acme",
		"11111111-1111-1111-1111-111111111111",
	)
	require.NoError(t, err)
	assert.False(t, member)
	assert.Empty(t, repositories)
	assert.False(t, listed)
}

func TestDirectoryUsesMainbranchAndFallsBackToMain(t *testing.T) {
	const accountID = "11111111-1111-1111-1111-111111111111"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer system-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/workspaces/acme/members/{11111111-1111-1111-1111-111111111111}":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"workspace":{"slug":"acme"}}`))
		case "/repositories/acme":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[
				{"uuid":"{22222222-2222-2222-2222-222222222222}","full_name":"acme/api","is_private":true,"mainbranch":{"name":"develop"}},
				{"uuid":"{33333333-3333-3333-3333-333333333333}","full_name":"acme/web","is_private":false}
			]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	directory := Directory{BaseURL: server.URL, HTTP: server.Client()}
	repositories, member, err := directory.VisibleRepositories(context.Background(), "system-token", "acme", accountID)
	require.NoError(t, err)
	require.True(t, member)
	require.Len(t, repositories, 2)
	assert.Equal(t, "acme/api", repositories[0].FullName)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", repositories[0].UUID)
	assert.Equal(t, "develop", repositories[0].DefaultBranch)
	assert.True(t, repositories[0].Private)
	assert.Equal(t, "acme", repositories[0].WorkspaceSlug)
	assert.Equal(t, "main", repositories[1].DefaultBranch)
	assert.False(t, repositories[1].Private)
}
