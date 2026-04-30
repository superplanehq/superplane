package members

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoveCommandCallsDelete(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.URL.Path == "/api/v1/users":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listUsersPayload))
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/organizations/"+testOrgID+"/users/user-1":
			deleted = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	ctx.Args = []string{"user-1"}
	email := ""

	require.NoError(t, (&removeCommand{email: &email}).Execute(ctx))
	require.True(t, deleted)
	require.Contains(t, stdout.String(), "Member removed: alice@example.com")
}

func TestRemoveCommandUUIDFastPathSkipsListCall(t *testing.T) {
	// Simulate: positional is a valid UUID. remove should skip the
	// /api/v1/users list call and go straight to DELETE.
	const userUUID = "11111111-2222-3333-4444-555555555555"
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.URL.Path == "/api/v1/users":
			t.Fatalf("unexpected list call; UUID fast path should skip it")
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/organizations/"+testOrgID+"/users/"+userUUID:
			deleted = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	ctx.Args = []string{userUUID}
	email := ""
	require.NoError(t, (&removeCommand{email: &email}).Execute(ctx))
	require.True(t, deleted)
	require.Contains(t, stdout.String(), "Member removed: "+userUUID)
}

func TestRemoveCommandRejectsBothPositionalAndEmail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me" {
			writeMeResponse(w)
			return
		}
		t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(server.Close)

	ctx, _ := newTestContext(t, server, "text")
	ctx.Args = []string{"user-1"}
	email := "alice@example.com"
	err := (&removeCommand{email: &email}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not both")
}

func TestRemoveCommandFailsOnUnknownMember(t *testing.T) {
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

	err := (&removeCommand{email: &email}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
