package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gh "github.com/google/go-github/v74/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GitHub__Setup(t *testing.T) {
	g := &GitHub{}

	t.Run("personal scope", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, g.Sync(core.SyncContext{Integration: integrationCtx}))

		//
		// Browser action is created
		//
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, integrationCtx.BrowserAction.Method, "POST")
		assert.NotEmpty(t, integrationCtx.BrowserAction.Description)
		assert.Equal(t, integrationCtx.BrowserAction.URL, "https://github.com/settings/apps/new")

		//
		// Metadata is set
		//
		require.NotNil(t, integrationCtx.Metadata)
		metadata := integrationCtx.Metadata.(Metadata)
		assert.Empty(t, metadata.Owner)
		assert.NotEmpty(t, metadata.State)
	})

	t.Run("organization scope", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		require.NoError(t, g.Sync(core.SyncContext{
			Configuration: Configuration{Organization: "testhq"},
			Integration:   integrationCtx,
		}))

		//
		// Browser action is created
		//
		require.NotNil(t, integrationCtx.BrowserAction)
		assert.Equal(t, integrationCtx.BrowserAction.Method, "POST")
		assert.NotEmpty(t, integrationCtx.BrowserAction.Description)
		assert.Equal(t, integrationCtx.BrowserAction.URL, "https://github.com/organizations/testhq/settings/apps/new")

		//
		// Metadata is set
		//
		require.NotNil(t, integrationCtx.Metadata)
		metadata := integrationCtx.Metadata.(Metadata)
		assert.Equal(t, metadata.Owner, "testhq")
		assert.NotEmpty(t, metadata.State)
	})
}

func Test__listInstallationRepositories__paginates_all_pages(t *testing.T) {
	t.Parallel()

	type reposResponse struct {
		TotalCount   int `json:"total_count"`
		Repositories []struct {
			ID      int64  `json:"id"`
			Name    string `json:"name"`
			HTMLURL string `json:"html_url"`
		} `json:"repositories"`
	}

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/installation/repositories", r.URL.Path)

		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}

		w.Header().Set("Content-Type", "application/json")

		switch page {
		case "1":
			// Provide Link header to instruct the client there is a next page.
			next := fmt.Sprintf(`<%s/installation/repositories?page=2&per_page=100>; rel="next", <%s/installation/repositories?page=2&per_page=100>; rel="last"`, srv.URL, srv.URL)
			w.Header().Set("Link", next)

			_ = json.NewEncoder(w).Encode(reposResponse{
				TotalCount: 2,
				Repositories: []struct {
					ID      int64  `json:"id"`
					Name    string `json:"name"`
					HTMLURL string `json:"html_url"`
				}{
					{ID: 1, Name: "repo1", HTMLURL: "https://github.com/test/repo1"},
				},
			})
		case "2":
			_ = json.NewEncoder(w).Encode(reposResponse{
				TotalCount: 2,
				Repositories: []struct {
					ID      int64  `json:"id"`
					Name    string `json:"name"`
					HTMLURL string `json:"html_url"`
				}{
					{ID: 2, Name: "repo2", HTMLURL: "https://github.com/test/repo2"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	client := gh.NewClient(srv.Client())
	baseURL, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL
	client.UploadURL = baseURL

	repos, err := listInstallationRepositories(context.Background(), client)
	require.NoError(t, err)
	require.Len(t, repos, 2)
	require.Equal(t, int64(1), repos[0].ID)
	require.Equal(t, "repo1", repos[0].Name)
	require.Equal(t, "https://github.com/test/repo1", repos[0].URL)
	require.Equal(t, int64(2), repos[1].ID)
	require.Equal(t, "repo2", repos[1].Name)
	require.Equal(t, "https://github.com/test/repo2", repos[1].URL)
}
