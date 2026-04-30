package groups

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const listGroupsPayload = `{
  "groups": [
    {
      "metadata": {"name":"engineers","domainType":"DOMAIN_TYPE_ORGANIZATION","domainId":"` + testOrgID + `"},
      "spec":     {"displayName":"Engineers","role":"org_admin"},
      "status":   {"membersCount":3}
    }
  ]
}`

func TestListCommandRendersText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups":
			require.Equal(t, "DOMAIN_TYPE_ORGANIZATION", r.URL.Query().Get("domainType"))
			require.Equal(t, testOrgID, r.URL.Query().Get("domainId"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listGroupsPayload))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	require.NoError(t, (&listCommand{}).Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "NAME")
	require.Contains(t, out, "engineers")
	require.Contains(t, out, "Engineers")
	require.Contains(t, out, "3")
}

func TestListCommandShowsEmptyState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			writeMeResponse(w)
		case "/api/v1/groups":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"groups":[]}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	require.NoError(t, (&listCommand{}).Execute(ctx))
	require.Contains(t, stdout.String(), "No groups found.")
}
