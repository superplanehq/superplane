package bitbucket

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAccountID       = "11111111-1111-1111-1111-111111111111"
	testPermissionsPath = "/workspaces/acme/permissions/repositories"
	testPermissionQuery = `user.uuid="{11111111-1111-1111-1111-111111111111}"`
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
	catalog, err := directory.CatalogVisibleTo(context.Background(), testAccountID, []InstallationRef{{ID: "install-1", WorkspaceSlug: "acme"}}, func(string) (string, error) { return "system-token", nil })
	require.NoError(t, err)
	repositories := catalog.Repositories
	assert.Empty(t, repositories)
	assert.Empty(t, catalog.Workspaces)
	assert.False(t, listed)
}

func TestDirectoryDetectsInstalledWorkspacesWithoutWritableRepositories(t *testing.T) {
	for _, role := range []string{"owner", "member", ""} {
		t.Run(role, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/workspaces/acme/permissions":
					assert.Equal(t, testPermissionQuery, r.URL.Query().Get("q"))
					if role != "" {
						_, _ = w.Write([]byte(`{"values":[{"permission":"` + role + `","workspace":{"slug":"acme"}}]}`))
						return
					}
					_, _ = w.Write([]byte(`{"values":[]}`))
				case testPermissionsPath, "/repositories/acme":
					_, _ = w.Write([]byte(`{"values":[]}`))
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)
			directory := Directory{BaseURL: server.URL, HTTP: server.Client()}
			catalog, err := directory.CatalogVisibleTo(context.Background(), testAccountID,
				[]InstallationRef{{ID: "installation-1", WorkspaceUUID: "workspace-1", WorkspaceSlug: "acme"}},
				func(string) (string, error) { return "system-token", nil })
			require.NoError(t, err)
			assert.Empty(t, catalog.Repositories)
			if role == "" {
				assert.Empty(t, catalog.Workspaces)
				return
			}
			require.Len(t, catalog.Workspaces, 1)
			assert.Equal(t, "acme", catalog.Workspaces[0].WorkspaceSlug)
			assert.Equal(t, "installation-1", catalog.Workspaces[0].ID)
		})
	}
}

func TestDirectoryKeepsAConfirmedWorkspaceWhenRepositoryListingFails(t *testing.T) {
	checkedOther := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/workspaces/acme/permissions":
			_, _ = w.Write([]byte(`{"values":[{"permission":"owner","workspace":{"slug":"acme"}}]}`))
		case "/workspaces/acme/permissions/repositories":
			_, _ = w.Write([]byte(`{"values":[]}`))
		case "/repositories/acme":
			http.Error(w, "unavailable", http.StatusInternalServerError)
		case "/workspaces/other/permissions", "/workspaces/other/permissions/repositories":
			checkedOther = true
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	directory := Directory{BaseURL: server.URL, HTTP: server.Client()}
	catalog, err := directory.CatalogVisibleTo(context.Background(), testAccountID, []InstallationRef{
		{ID: "install-confirmed", WorkspaceUUID: "workspace-1", WorkspaceSlug: "acme"},
		{ID: "install-hidden", WorkspaceSlug: "other"},
	}, func(string) (string, error) { return "system-token", nil })
	require.NoError(t, err)
	require.True(t, checkedOther)
	require.Len(t, catalog.Workspaces, 1)
	assert.Equal(t, "install-confirmed", catalog.Workspaces[0].ID)
	assert.Equal(t, "acme", catalog.Workspaces[0].WorkspaceSlug)
	assert.Empty(t, catalog.Repositories)
}

func TestDirectoryReportsUnavailableTokensInsteadOfMissingInstallations(t *testing.T) {
	tokenErr := errors.New("expired token")
	_, err := (Directory{}).CatalogVisibleTo(context.Background(), testAccountID,
		[]InstallationRef{{ID: "installation-1", WorkspaceSlug: "acme"}},
		func(string) (string, error) { return "", tokenErr })
	require.ErrorIs(t, err, tokenErr)
}

func TestDirectoryHidesRepositoriesWithoutAPushPermission(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer system-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case testPermissionsPath:
			assert.Equal(t, testPermissionQuery, r.URL.Query().Get("q"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[
				{"type":"repository_permission","permission":"admin","repository":{"uuid":"{22222222-2222-2222-2222-222222222222}","full_name":"acme/api"}},
				{"type":"repository_permission","permission":"read","repository":{"uuid":"{33333333-3333-3333-3333-333333333333}","full_name":"acme/web"}}
			]}`))
		case "/repositories/acme":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[
				{"uuid":"{22222222-2222-2222-2222-222222222222}","full_name":"acme/api","is_private":true,"mainbranch":{"name":"develop"}},
				{"uuid":"{33333333-3333-3333-3333-333333333333}","full_name":"acme/web","is_private":false},
				{"uuid":"{44444444-4444-4444-4444-444444444444}","full_name":"acme/secret","is_private":true}
			]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	directory := Directory{BaseURL: server.URL, HTTP: server.Client()}
	catalog, err := directory.CatalogVisibleTo(context.Background(), testAccountID, []InstallationRef{{ID: "install-1", WorkspaceSlug: "acme"}}, func(string) (string, error) { return "system-token", nil })
	require.NoError(t, err)
	repositories := catalog.Repositories
	require.Len(t, repositories, 1)
	assert.Equal(t, "acme/api", repositories[0].FullName)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", repositories[0].UUID)
	assert.Equal(t, "develop", repositories[0].DefaultBranch)
	assert.True(t, repositories[0].Private)
	assert.Equal(t, "acme", repositories[0].WorkspaceSlug)
}

func TestDirectoryDoesNotListRepositoriesForAMemberWithoutGrants(t *testing.T) {
	listed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case testPermissionsPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"values":[]}`))
		case "/repositories/acme":
			listed = true
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	directory := Directory{BaseURL: server.URL, HTTP: server.Client()}
	catalog, err := directory.CatalogVisibleTo(context.Background(), testAccountID, []InstallationRef{{ID: "install-1", WorkspaceSlug: "acme"}}, func(string) (string, error) { return "system-token", nil })
	require.NoError(t, err)
	repositories := catalog.Repositories
	assert.Empty(t, repositories)
	assert.False(t, listed)
}

func TestDirectoryListsEveryRepositoryForAWorkspaceOwner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer system-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case testPermissionsPath:
			_, _ = w.Write([]byte(`{"values":[]}`))
		case "/workspaces/acme/permissions":
			assert.Equal(t, testPermissionQuery, r.URL.Query().Get("q"))
			_, _ = w.Write([]byte(`{"values":[{"permission":"owner","workspace":{"slug":"acme"}}]}`))
		case "/workspaces/acme/members":
			_, _ = w.Write([]byte(`{"values":[{"user":{"uuid":"{11111111-1111-1111-1111-111111111111}"},"workspace":{"slug":"acme"}}]}`))
		case "/repositories/acme":
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
	catalog, err := directory.CatalogVisibleTo(context.Background(), testAccountID, []InstallationRef{{ID: "install-1", WorkspaceSlug: "acme"}}, func(string) (string, error) { return "system-token", nil })
	require.NoError(t, err)
	repositories := catalog.Repositories
	require.Len(t, repositories, 2)
	assert.Equal(t, "acme/api", repositories[0].FullName)
	assert.Equal(t, "acme/web", repositories[1].FullName)
	assert.Equal(t, "acme", repositories[0].WorkspaceSlug)
}

func TestDirectoryFollowsPermissionPagesAndFallsBackToMain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == testPermissionsPath && r.URL.Query().Get("page") == "2":
			_, _ = w.Write([]byte(`{"values":[
				{"type":"repository_permission","permission":"write","repository":{"uuid":"{33333333-3333-3333-3333-333333333333}","full_name":"acme/web"}}
			]}`))
		case r.URL.Path == testPermissionsPath:
			_, _ = w.Write([]byte(`{"values":[
				{"type":"repository_permission","permission":"admin","repository":{"uuid":"{22222222-2222-2222-2222-222222222222}","full_name":"acme/api"}}
			],"next":"` + testPermissionsPath + `?page=2"}`))
		case r.URL.Path == "/repositories/acme":
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
	catalog, err := directory.CatalogVisibleTo(context.Background(), testAccountID, []InstallationRef{{ID: "install-1", WorkspaceSlug: "acme"}}, func(string) (string, error) { return "system-token", nil })
	require.NoError(t, err)
	repositories := catalog.Repositories
	require.Len(t, repositories, 2)
	assert.Equal(t, "acme/api", repositories[0].FullName)
	assert.Equal(t, "develop", repositories[0].DefaultBranch)
	assert.Equal(t, "acme/web", repositories[1].FullName)
	assert.Equal(t, "main", repositories[1].DefaultBranch)
	assert.False(t, repositories[1].Private)
}

func TestDirectoryResolvesTheWorkspaceSlugFromAGrant(t *testing.T) {
	const workspaceUUID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/workspaces/{" + workspaceUUID + "}/permissions/repositories":
			_, _ = w.Write([]byte(`{"values":[
				{"type":"repository_permission","permission":"write","repository":{"uuid":"{22222222-2222-2222-2222-222222222222}","full_name":"acme/api"}}
			]}`))
		case "/repositories/acme":
			_, _ = w.Write([]byte(`{"values":[
				{"uuid":"{22222222-2222-2222-2222-222222222222}","full_name":"acme/api","is_private":true}
			]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	directory := Directory{BaseURL: server.URL, HTTP: server.Client()}
	catalog, err := directory.CatalogVisibleTo(context.Background(), testAccountID, []InstallationRef{{ID: "install-1", WorkspaceSlug: workspaceUUID}}, func(string) (string, error) { return "system-token", nil })
	require.NoError(t, err)
	repositories := catalog.Repositories
	require.Len(t, repositories, 1)
	assert.Equal(t, "acme", repositories[0].WorkspaceSlug)
}
