package common

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

func Test__Client__ListIssueComments(t *testing.T) {
	t.Run("returns comments oldest first and follows pagination", func(t *testing.T) {
		var requests []*http.Request

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r)
			assert.Equal(t, "/repos/acme/payments/issues/12/comments", r.URL.Path)
			assert.Equal(t, "created", r.URL.Query().Get("sort"))
			assert.Equal(t, "asc", r.URL.Query().Get("direction"))

			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("page") == "2" {
				_, _ = w.Write([]byte(`[{"id": 2, "body": "Let's cap retries at 3.", "user": {"login": "bruno"}, "created_at": "2026-08-02T09:00:00Z"}]`))
				return
			}

			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/acme/payments/issues/12/comments?page=2>; rel="next"`, r.Host))
			_, _ = w.Write([]byte(`[{"id": 1, "body": "Confirmed on staging.", "user": {"login": "ana"}, "created_at": "2026-08-01T10:00:00Z"}]`))
		}))
		t.Cleanup(srv.Close)

		client := clientForTestServer(t, srv)

		comments, err := client.ListIssueComments(context.Background(), "acme/payments", 12)
		require.NoError(t, err)
		require.Len(t, requests, 2)

		require.Len(t, comments, 2)
		assert.Equal(t, "ana", comments[0].GetUser().GetLogin())
		assert.Equal(t, "Confirmed on staging.", comments[0].GetBody())
		assert.Equal(t, "bruno", comments[1].GetUser().GetLogin())
		assert.Equal(t, "Let's cap retries at 3.", comments[1].GetBody())
	})

	t.Run("propagates errors", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(srv.Close)

		client := clientForTestServer(t, srv)

		_, err := client.ListIssueComments(context.Background(), "acme/payments", 12)
		require.Error(t, err)
	})
}

// clientForTestServer builds a Client whose underlying go-github SDK client
// targets an httptest server, so tests can drive Client methods against a
// controlled HTTP fixture without a real GitHub API call.
func clientForTestServer(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()

	baseURL, err := url.Parse(srv.URL + "/")
	require.NoError(t, err)

	underlying := gh.NewClient(srv.Client())
	underlying.BaseURL = baseURL

	return &Client{owner: "acme", underlying: underlying}
}
