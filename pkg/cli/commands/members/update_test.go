package members

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type assignBody struct {
	DomainType string `json:"domainType"`
	DomainID   string `json:"domainId"`
	UserID     string `json:"userId"`
	UserEmail  string `json:"userEmail"`
}

func newAssignRoleServer(t *testing.T, seen *assignBody) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/roles/org_admin/users":
			payload, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(payload, seen))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestUpdateCommandAssignsRoleByID(t *testing.T) {
	var seen assignBody
	server := newAssignRoleServer(t, &seen)

	ctx, stdout := newTestContext(t, server, "text")
	role := "org_admin"
	email := ""
	cmd := &updateCommand{email: &email, role: &role}

	cobraCmd := ctx.Cmd
	cobraCmd.Flags().String("role", role, "")
	require.NoError(t, cobraCmd.Flags().Set("role", role))
	ctx.Args = []string{"user-2"}

	require.NoError(t, cmd.Execute(ctx))
	require.Equal(t, "DOMAIN_TYPE_ORGANIZATION", seen.DomainType)
	require.Equal(t, testOrgID, seen.DomainID)
	require.Equal(t, "user-2", seen.UserID)
	require.Empty(t, seen.UserEmail)
	require.Contains(t, stdout.String(), `Role "org_admin" assigned to user-2`)
}

func TestUpdateCommandAssignsRoleByEmailFlag(t *testing.T) {
	var seen assignBody
	server := newAssignRoleServer(t, &seen)

	ctx, _ := newTestContext(t, server, "text")
	role := "org_admin"
	email := "alice@example.com"
	cmd := &updateCommand{email: &email, role: &role}

	cobraCmd := ctx.Cmd
	cobraCmd.Flags().String("role", role, "")
	require.NoError(t, cobraCmd.Flags().Set("role", role))

	require.NoError(t, cmd.Execute(ctx))
	require.Empty(t, seen.UserID)
	require.Equal(t, "alice@example.com", seen.UserEmail)
}

func TestUpdateCommandAssignsRoleByEmailPositional(t *testing.T) {
	var seen assignBody
	server := newAssignRoleServer(t, &seen)

	ctx, stdout := newTestContext(t, server, "text")
	role := "org_admin"
	email := ""
	cmd := &updateCommand{email: &email, role: &role}

	cobraCmd := ctx.Cmd
	cobraCmd.Flags().String("role", role, "")
	require.NoError(t, cobraCmd.Flags().Set("role", role))
	// Positional with "@" must be forwarded as userEmail so the backend
	// (which uuid.Parses userId) does not reject the request.
	ctx.Args = []string{"carol@example.com"}

	require.NoError(t, cmd.Execute(ctx))
	require.Empty(t, seen.UserID)
	require.Equal(t, "carol@example.com", seen.UserEmail)
	require.Contains(t, stdout.String(), "assigned to carol@example.com")
}
