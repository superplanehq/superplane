package roles

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const listRolesPayload = `{
  "roles": [
    {
      "metadata": {"name":"org_admin","domainType":"DOMAIN_TYPE_ORGANIZATION","domainId":"` + testOrgID + `"},
      "spec":     {"displayName":"Admin","description":"Full access","permissions":[{"resource":"members","action":"update"}]}
    },
    {
      "metadata": {"name":"org_viewer"},
      "spec":     {"displayName":"Viewer"}
    }
  ]
}`

func TestListCommandRendersText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/roles":
			require.Equal(t, "DOMAIN_TYPE_ORGANIZATION", r.URL.Query().Get("domainType"))
			require.Equal(t, testOrgID, r.URL.Query().Get("domainId"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listRolesPayload))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	require.NoError(t, (&listCommand{}).Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "NAME")
	require.Contains(t, out, "org_admin")
	require.Contains(t, out, "Admin")
}

func TestGetCommandRendersText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/roles/org_admin":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"role":{"metadata":{"name":"org_admin"},"spec":{"displayName":"Admin","permissions":[{"resource":"members","action":"update"}]}}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	ctx.Args = []string{"org_admin"}
	require.NoError(t, (&getCommand{}).Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "Name: org_admin")
	require.Contains(t, out, "- members:update")
}
