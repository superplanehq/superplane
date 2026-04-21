package members

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

func newResolverServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/me":
			writeMeResponse(w)
		case "/api/v1/users":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listUsersPayload))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestResolveMemberByID(t *testing.T) {
	server := newResolverServer(t)
	ctx, _ := newTestContext(t, server, "text")

	user, err := resolveMember(ctx, testOrgID, "user-2", "")
	require.NoError(t, err)
	metadata := user.GetMetadata()
	require.Equal(t, "user-2", metadata.GetId())
	require.Equal(t, "bob@example.com", metadata.GetEmail())
}

func TestResolveMemberByEmail(t *testing.T) {
	server := newResolverServer(t)
	ctx, _ := newTestContext(t, server, "text")

	user, err := resolveMember(ctx, testOrgID, "", "ALICE@example.com")
	require.NoError(t, err)
	metadata := user.GetMetadata()
	require.Equal(t, "user-1", metadata.GetId())
}

func TestResolveMemberPrefersPositional(t *testing.T) {
	server := newResolverServer(t)
	ctx, _ := newTestContext(t, server, "text")

	// Positional is "bob@example.com" — resolver accepts either id or email
	// in the positional slot and should find the match by email.
	user, err := resolveMember(ctx, testOrgID, "bob@example.com", "")
	require.NoError(t, err)
	metadata := user.GetMetadata()
	require.Equal(t, "user-2", metadata.GetId())
}

func TestResolveMemberNotFound(t *testing.T) {
	server := newResolverServer(t)
	ctx, _ := newTestContext(t, server, "text")

	_, err := resolveMember(ctx, testOrgID, "ghost@example.com", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestResolveMemberRequiresIdentifier(t *testing.T) {
	ctx, _ := newTestContext(t, nil, "text")
	_, err := resolveMember(ctx, testOrgID, "", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "user id or --email")
}

func TestResolveMemberRejectsBothPositionalAndEmail(t *testing.T) {
	ctx, _ := newTestContext(t, nil, "text")
	_, err := resolveMember(ctx, testOrgID, "user-1", "alice@example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not both")
}

func TestResolveMemberTrimsWhitespaceOnID(t *testing.T) {
	server := newResolverServer(t)
	ctx, _ := newTestContext(t, server, "text")

	user, err := resolveMember(ctx, testOrgID, "  user-2  ", "")
	require.NoError(t, err)
	metadata := user.GetMetadata()
	require.Equal(t, "user-2", metadata.GetId())
}

func TestSplitUserIdentifierAmbiguous(t *testing.T) {
	_, _, err := core.SplitUserIdentifier("user-1", "alice@example.com")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not both")
}

func TestSplitUserIdentifierEmailPositional(t *testing.T) {
	id, email, err := core.SplitUserIdentifier("alice@example.com", "")
	require.NoError(t, err)
	require.Empty(t, id)
	require.Equal(t, "alice@example.com", email)
}
