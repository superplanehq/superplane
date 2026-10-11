package bitbucket

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func statusCheckIntegration() *contexts.IntegrationContext {
	return workspaceTokenIntegration()
}

func Test__Bitbucket__ListResources__StatusCheck(t *testing.T) {
	t.Run("discovers checks from recent PR statuses", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			bitbucketResponse(http.StatusOK, `{"values": [
				{"source": {"commit": {"hash": "aaa111aaa111aaa111aaa111aaa111aaa111aaaa"}}},
				{"source": {"commit": {"hash": "bbb222bbb222bbb222bbb222bbb222bbb222bbbb"}}}
			]}`),
			bitbucketResponse(http.StatusOK, `{"values": [
				{"key": "build-a", "name": "Build A", "state": "SUCCESSFUL", "url": "https://ci.example/a", "updated_on": "2026-04-22T10:05:00+00:00"}
			]}`),
			bitbucketResponse(http.StatusOK, `{"values": [
				{"key": "build-b", "name": "Build B", "state": "FAILED", "url": "https://ci.example/b", "updated_on": "2026-04-22T10:06:00+00:00"}
			]}`),
		}}

		resources, err := (&Bitbucket{}).ListResources("status_check", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: statusCheckIntegration(),
			Parameters:  map[string]string{"repository": "acme/widgets"},
		})
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "status_check", resources[0].Type)
		assert.Equal(t, "build-a", resources[0].ID)
		assert.Equal(t, "Build A", resources[0].Name)
		assert.Equal(t, "https://ci.example/a", resources[0].URL)
		assert.Equal(t, "build-b", resources[1].ID)

		require.NotEmpty(t, httpCtx.Requests)
		first := httpCtx.Requests[0]
		assert.Equal(t, "/2.0/repositories/acme/widgets/pullrequests", first.URL.Path)
		assert.Contains(t, first.URL.RawQuery, "sort=-updated_on")
		assert.Contains(t, first.URL.RawQuery, "pagelen=3")
	})

	t.Run("deduplicates duplicate SHAs and paginates statuses", func(t *testing.T) {
		sha := "aaa111aaa111aaa111aaa111aaa111aaa111aaaa"
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			bitbucketResponse(http.StatusOK, `{"values": [
				{"source": {"commit": {"hash": "`+sha+`"}}},
				{"source": {"commit": {"hash": "`+strings.ToUpper(sha)+`"}}}
			]}`),
			bitbucketResponse(http.StatusOK, `{"values": [
				{"key": "build-a", "name": "Build A", "state": "SUCCESSFUL", "updated_on": "2026-04-22T10:05:00+00:00"}
			], "next": "https://api.bitbucket.org/2.0/next-page"}`),
			bitbucketResponse(http.StatusOK, `{"values": [
				{"key": "BUILD-A", "name": "Build A renamed", "state": "SUCCESSFUL", "url": "https://ci.example/a2", "updated_on": "2026-04-22T11:00:00+00:00"}
			]}`),
		}}

		resources, err := (&Bitbucket{}).ListResources("status_check", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: statusCheckIntegration(),
			Parameters:  map[string]string{"repository": "acme/widgets"},
		})
		require.NoError(t, err)
		require.Len(t, resources, 1)
		// Newest observed metadata wins, deduped by case-insensitive key
		assert.Equal(t, "BUILD-A", resources[0].ID)
		assert.Equal(t, "Build A renamed", resources[0].Name)
		// Only one ListCommitStatuses call for duplicate SHAs (plus pagination)
		assert.Len(t, httpCtx.Requests, 3)
	})

	t.Run("falls back to default branch head when PRs have no statuses", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			bitbucketResponse(http.StatusOK, `{"values": [{"source": {"commit": {"hash": "aaa111aaa111aaa111aaa111aaa111aaa111aaaa"}}}]}`),
			bitbucketResponse(http.StatusOK, `{"values": []}`),
			bitbucketResponse(http.StatusOK, `{"full_name":"acme/widgets","mainbranch":{"name":"main"}}`),
			bitbucketResponse(http.StatusOK, `{"name":"main","target":{"hash":"ddd444ddd444ddd444ddd444ddd444ddd444dddd"}}`),
			bitbucketResponse(http.StatusOK, `{"values": [
				{"key": "main-build", "name": "Main Build", "state": "SUCCESSFUL", "url": "https://ci.example/main"}
			]}`),
		}}

		resources, err := (&Bitbucket{}).ListResources("status_check", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: statusCheckIntegration(),
			Parameters:  map[string]string{"repository": "acme/widgets"},
		})
		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "main-build", resources[0].ID)
		assert.Equal(t, "Main Build", resources[0].Name)
	})

	t.Run("empty repository without default branch returns empty catalog", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			bitbucketResponse(http.StatusOK, `{"values": []}`),
			bitbucketResponse(http.StatusOK, `{"full_name":"acme/widgets"}`),
		}}

		resources, err := (&Bitbucket{}).ListResources("status_check", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: statusCheckIntegration(),
			Parameters:  map[string]string{"repository": "acme/widgets"},
		})
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("skips blank keys and falls back to key for name", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			bitbucketResponse(http.StatusOK, `{"values": [{"source": {"commit": {"hash": "aaa111aaa111aaa111aaa111aaa111aaa111aaaa"}}}]}`),
			bitbucketResponse(http.StatusOK, `{"values": [
				{"key": "", "name": "No Key", "state": "SUCCESSFUL"},
				{"key": "only-key", "name": "", "state": "SUCCESSFUL", "url": "https://ci.example/k"},
				{"key": "build-a", "name": "Build A", "state": "SUCCESSFUL"}
			]}`),
		}}

		resources, err := (&Bitbucket{}).ListResources("status_check", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: statusCheckIntegration(),
			Parameters:  map[string]string{"repository": "acme/widgets"},
		})
		require.NoError(t, err)
		require.Len(t, resources, 2)
		// Sorted consistently by key
		assert.Equal(t, "build-a", resources[0].ID)
		assert.Equal(t, "only-key", resources[1].ID)
		assert.Equal(t, "only-key", resources[1].Name)
	})

	t.Run("two keys can share one display name", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			bitbucketResponse(http.StatusOK, `{"values": [{"source": {"commit": {"hash": "aaa111aaa111aaa111aaa111aaa111aaa111aaaa"}}}]}`),
			bitbucketResponse(http.StatusOK, `{"values": [
				{"key": "e2e-1", "name": "E2E", "state": "SUCCESSFUL"},
				{"key": "e2e-2", "name": "E2E", "state": "FAILED"}
			]}`),
		}}

		resources, err := (&Bitbucket{}).ListResources("status_check", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: statusCheckIntegration(),
			Parameters:  map[string]string{"repository": "acme/widgets"},
		})
		require.NoError(t, err)
		require.Len(t, resources, 2)
		assert.Equal(t, "e2e-1", resources[0].ID)
		assert.Equal(t, "e2e-2", resources[1].ID)
		assert.Equal(t, "E2E", resources[0].Name)
		assert.Equal(t, "E2E", resources[1].Name)
	})

	t.Run("rejects repository outside workspace", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		_, err := (&Bitbucket{}).ListResources("status_check", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: statusCheckIntegration(),
			Parameters:  map[string]string{"repository": "other/widgets"},
		})
		require.ErrorContains(t, err, "not accessible to workspace")
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("surfaces API failures instead of empty catalog", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"expired"}}`))},
		}}
		_, err := (&Bitbucket{}).ListResources("status_check", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: statusCheckIntegration(),
			Parameters:  map[string]string{"repository": "acme/widgets"},
		})
		require.Error(t, err)
	})

	t.Run("repository-scoped token uses configured repository", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
			bitbucketResponse(http.StatusOK, `{"values": []}`),
			bitbucketResponse(http.StatusOK, `{"full_name":"acme/widgets"}`),
		}}
		integration := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "repo-token"},
			Metadata: Metadata{
				AuthType:  AuthTypeRepositoryAccessToken,
				Workspace: &WorkspaceMetadata{Slug: "acme"},
				Repository: &RepositoryMetadata{
					FullName: "acme/widgets",
					Slug:     "widgets",
					Name:     "widgets",
				},
			},
		}
		resources, err := (&Bitbucket{}).ListResources("status_check", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integration,
			Parameters:  map[string]string{},
		})
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("repository-scoped token rejects other repository", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{}
		integration := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "repo-token"},
			Metadata: Metadata{
				AuthType:  AuthTypeRepositoryAccessToken,
				Workspace: &WorkspaceMetadata{Slug: "acme"},
				Repository: &RepositoryMetadata{
					FullName: "acme/widgets",
					Slug:     "widgets",
					Name:     "widgets",
				},
			},
		}
		_, err := (&Bitbucket{}).ListResources("status_check", core.ListResourcesContext{
			HTTP:        httpCtx,
			Integration: integration,
			Parameters:  map[string]string{"repository": "acme/other"},
		})
		require.ErrorContains(t, err, "not accessible")
	})
}

func Test__Client__GetBranchHead(t *testing.T) {
	client, httpCtx := stubBitbucketClient(bitbucketResponse(http.StatusOK, `{"name":"main","target":{"hash":"abc123"}}`))
	head, err := client.GetBranchHead("acme/widgets", "main")
	require.NoError(t, err)
	assert.Equal(t, "abc123", head)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "/2.0/repositories/acme/widgets/refs/branches/main", httpCtx.Requests[0].URL.Path)
}
