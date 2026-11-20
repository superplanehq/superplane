package common

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientListHumanReviewers(t *testing.T) {
	t.Run("paginates eligible humans and excludes bots and read access", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/repos/acme/payments/collaborators", r.URL.Path)
			require.Equal(t, "push", r.URL.Query().Get("permission"))
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("page") == "2" {
				_, _ = w.Write([]byte(`[{"id":2,"login":"maintainer","type":"User","permissions":{"maintain":true}},{"id":1,"login":"ANA","type":"User","permissions":{"push":true}}]`))
				return
			}
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/acme/payments/collaborators?page=2>; rel="next"`, r.Host))
			_, _ = w.Write([]byte(`[{"id":1,"login":"ana","type":"User","permissions":{"push":true}},{"login":"reader","type":"User","permissions":{"pull":true}},{"login":"agent[bot]","type":"User","permissions":{"push":true}},{"login":"bot","type":"Bot","permissions":{"admin":true}},{"login":"unknown","permissions":{"push":true}}]`))
		}))
		t.Cleanup(server.Close)
		users, err := clientForTestServer(t, server).ListHumanReviewers(context.Background(), "acme/payments")
		require.NoError(t, err)
		require.Len(t, users, 2)
		require.Equal(t, "ana", users[0].GetLogin())
		require.Equal(t, "maintainer", users[1].GetLogin())
	})
	t.Run("a failed page fails the complete lookup", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("page") == "2" {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"message":"Forbidden"}`))
				return
			}
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/acme/payments/collaborators?page=2>; rel="next"`, r.Host))
			_, _ = w.Write([]byte(`[{"id":1,"login":"ana","type":"User","permissions":{"push":true}}]`))
		}))
		t.Cleanup(server.Close)
		users, err := clientForTestServer(t, server).ListHumanReviewers(context.Background(), "acme/payments")
		require.Error(t, err)
		require.Nil(t, users)
	})
}
