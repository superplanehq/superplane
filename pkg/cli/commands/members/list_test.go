package members

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const listUsersPayload = `{
  "users": [
    {
      "metadata": {"id":"user-1","email":"alice@example.com"},
      "spec":     {"displayName":"Alice"},
      "status":   {"roles":[{"roleName":"org_owner"}]}
    },
    {
      "metadata": {"id":"user-2","email":"bob@example.com"},
      "spec":     {"displayName":"Bob"},
      "status":   {"roles":[{"roleName":"org_viewer"}]}
    }
  ]
}`

func newListUsersServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users":
			require.Equal(t, "DOMAIN_TYPE_ORGANIZATION", r.URL.Query().Get("domainType"))
			require.Equal(t, testOrgID, r.URL.Query().Get("domainId"))
			require.Equal(t, "true", r.URL.Query().Get("includeRoles"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listUsersPayload))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestListCommandRendersText(t *testing.T) {
	server := newListUsersServer(t)
	ctx, stdout := newTestContext(t, server, "text")

	err := (&listCommand{}).Execute(ctx)
	require.NoError(t, err)

	out := stdout.String()
	require.Contains(t, out, "ID")
	require.Contains(t, out, "EMAIL")
	require.Contains(t, out, "ROLES")
	require.Contains(t, out, "alice@example.com")
	require.Contains(t, out, "org_owner")
	require.Contains(t, out, "bob@example.com")
}

func TestListCommandRendersJSON(t *testing.T) {
	server := newListUsersServer(t)
	ctx, stdout := newTestContext(t, server, "json")

	err := (&listCommand{}).Execute(ctx)
	require.NoError(t, err)

	out := stdout.String()
	require.Contains(t, out, `"email": "alice@example.com"`)
	require.Contains(t, out, `"displayName": "Bob"`)
}

func TestListCommandShowsEmptyState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			writeMeResponse(w)
		case "/api/v1/users":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"users":[]}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	err := (&listCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "No members found.")
}
