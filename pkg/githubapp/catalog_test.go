package githubapp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gh "github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatalogGitHubRequestsFollowOnboardingGuide(t *testing.T) {
	t.Run("lists App installations with App authentication pagination", func(t *testing.T) {
		client, requests := guideClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/app/installations", r.URL.Path)
			assert.Equal(t, "100", r.URL.Query().Get("per_page"))
			page := r.URL.Query().Get("page")
			if page == "" {
				w.Header().Set("Link", fmt.Sprintf(`<%s/app/installations?per_page=100&page=2>; rel="next"`, requestsURL(r)))
				_, _ = w.Write([]byte(`[{"id":101,"account":{"id":1,"login":"acme"},"html_url":"https://github.com/organizations/acme/settings/installations/101"}]`))
				return
			}
			assert.Equal(t, "2", page)
			_, _ = w.Write([]byte(`[{"id":202,"account":{"id":2,"login":"example"}}]`))
		})

		catalog := &Catalog{appClient: client}
		installations, err := catalog.listInstallations(context.Background())
		require.NoError(t, err)
		assert.Equal(t, []int64{101, 202}, []int64{installations[0].GetID(), installations[1].GetID()})
		assert.Equal(t, "https://github.com/organizations/acme/settings/installations/101", installations[0].GetHTMLURL())
		assert.Equal(t, 2, *requests)
	})

	t.Run("lists installation requests with App authentication", func(t *testing.T) {
		client, _ := guideClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/app/installation-requests", r.URL.Path)
			assert.Equal(t, "100", r.URL.Query().Get("per_page"))
			_, _ = w.Write([]byte(`[{"id":44,"account":{"id":2,"login":"acme"},"requester":{"id":9,"login":"member"}}]`))
		})

		catalog := &Catalog{appClient: client}
		requests, err := catalog.listInstallationRequests(context.Background())
		require.NoError(t, err)
		require.Len(t, requests, 1)
		assert.Equal(t, int64(44), requests[0].GetID())
	})

	t.Run("lists installation repositories with an installation client", func(t *testing.T) {
		client, _ := guideClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/installation/repositories", r.URL.Path)
			assert.Equal(t, "100", r.URL.Query().Get("per_page"))
			_, _ = w.Write([]byte(`{"total_count":1,"repositories":[{"id":77,"full_name":"acme/api","default_branch":"main"}]}`))
		})

		catalog := &Catalog{installation: func(installationID int64) (*gh.Client, error) {
			assert.Equal(t, int64(101), installationID)
			return client, nil
		}}
		repositories, err := catalog.listRepositories(context.Background(), 101)
		require.NoError(t, err)
		require.Len(t, repositories, 1)
		assert.Equal(t, int64(77), repositories[0].GetID())
	})

	t.Run("keeps only push-capable collaborators", func(t *testing.T) {
		client, _ := guideClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/repos/acme/api/collaborators", r.URL.Path)
			assert.Equal(t, "100", r.URL.Query().Get("per_page"))
			_, _ = w.Write([]byte(`[
          {"id":9,"login":"writer","permissions":{"push":true}},
          {"id":10,"login":"reader","permissions":{"push":false}}
        ]`))
		})

		collaborators, err := listCollaborators(context.Background(), client, "acme", "api")
		require.NoError(t, err)
		models := collaboratorModels(77, collaborators)
		require.Len(t, models, 1)
		assert.Equal(t, int64(9), models[0].GitHubUserID)
		assert.Equal(t, "writer", models[0].GitHubLogin)
	})
}

func guideClient(t *testing.T, handler http.HandlerFunc) (*gh.Client, *int) {
	t.Helper()
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	client := gh.NewClient(server.Client())
	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = baseURL
	client.UploadURL = baseURL
	return client, &requests
}

func requestsURL(r *http.Request) string {
	return "http://" + r.Host
}
