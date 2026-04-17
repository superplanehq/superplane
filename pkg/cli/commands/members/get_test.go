package members

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetCommandRendersMemberText(t *testing.T) {
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

	ctx, stdout := newTestContext(t, server, "text")
	ctx.Args = []string{"user-1"}
	email := ""
	require.NoError(t, (&getCommand{email: &email}).Execute(ctx))

	out := stdout.String()
	require.Contains(t, out, "ID: user-1")
	require.Contains(t, out, "Email: alice@example.com")
	require.Contains(t, out, "Name: Alice")
	require.Contains(t, out, "org_owner")
}

func TestGetCommandByEmailFlag(t *testing.T) {
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

	ctx, stdout := newTestContext(t, server, "text")
	email := "bob@example.com"
	require.NoError(t, (&getCommand{email: &email}).Execute(ctx))
	require.Contains(t, stdout.String(), "ID: user-2")
}

func TestGetCommandNotFound(t *testing.T) {
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

	ctx, _ := newTestContext(t, server, "text")
	ctx.Args = []string{"ghost@example.com"}
	email := ""
	err := (&getCommand{email: &email}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
