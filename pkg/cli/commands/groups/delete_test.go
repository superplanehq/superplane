package groups

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeleteCommand(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/groups/engineers":
			require.Equal(t, "DOMAIN_TYPE_ORGANIZATION", r.URL.Query().Get("domainType"))
			require.Equal(t, testOrgID, r.URL.Query().Get("domainId"))
			deleted = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	ctx.Args = []string{"engineers"}
	require.NoError(t, (&deleteCommand{}).Execute(ctx))
	require.True(t, deleted)
	require.Contains(t, stdout.String(), "Group deleted: engineers")
}
