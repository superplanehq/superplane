package members

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const listInvitationsPayload = `{
  "invitations": [
    {"id":"inv-1","email":"bob@example.com","state":"pending"}
  ]
}`

func TestInvitationsListRendersText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organizations/"+testOrgID+"/invitations":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listInvitationsPayload))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	require.NoError(t, (&invitationsListCommand{}).Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "EMAIL")
	require.Contains(t, out, "bob@example.com")
	require.Contains(t, out, "pending")
}

func TestInvitationsCreateSendsEmail(t *testing.T) {
	var body struct {
		Email string `json:"email"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/organizations/"+testOrgID+"/invitations":
			payload, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(payload, &body))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"invitation":{"id":"inv-9","email":"new@example.com","state":"pending"}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	email := "new@example.com"
	ctx, stdout := newTestContext(t, server, "text")
	require.NoError(t, (&invitationsCreateCommand{email: &email}).Execute(ctx))
	require.Equal(t, "new@example.com", body.Email)
	require.Contains(t, stdout.String(), "inv-9")
}

func TestInvitationsRemoveCallsDelete(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/organizations/"+testOrgID+"/invitations/inv-1":
			deleted = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	ctx.Args = []string{"inv-1"}
	require.NoError(t, (&invitationsRemoveCommand{}).Execute(ctx))
	require.True(t, deleted)
	require.Contains(t, stdout.String(), "Invitation removed: inv-1")
}
