package members

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const inviteLinkPayload = `{"inviteLink":{"id":"link-1","organizationId":"` + testOrgID + `","token":"abc","enabled":true}}`

func TestInviteLinkGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/organizations/"+testOrgID+"/invite-link":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(inviteLinkPayload))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	require.NoError(t, (&inviteLinkGetCommand{}).Execute(ctx))
	out := stdout.String()
	require.Contains(t, out, "Enabled: true")
	require.Contains(t, out, "Token: abc")
}

func TestInviteLinkUpdateSendsEnabled(t *testing.T) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.URL.Path == "/api/v1/organizations/"+testOrgID+"/invite-link":
			require.Equal(t, http.MethodPatch, r.Method)
			payload, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(payload, &body))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"inviteLink":{"id":"link-1","enabled":false}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := newTestContext(t, server, "text")
	// Simulate --enabled=false being set.
	enabled := false
	cobraCmd := ctx.Cmd
	cobraCmd.Flags().Bool("enabled", enabled, "")
	require.NoError(t, cobraCmd.Flags().Set("enabled", "false"))

	require.NoError(t, (&inviteLinkUpdateCommand{enabled: &enabled}).Execute(ctx))
	require.False(t, body.Enabled)
}

func TestInviteLinkReset(t *testing.T) {
	reset := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/organizations/"+testOrgID+"/invite-link/reset":
			reset = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"inviteLink":{"id":"link-2","token":"new","enabled":true}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	require.NoError(t, (&inviteLinkResetCommand{}).Execute(ctx))
	require.True(t, reset)
	require.Contains(t, stdout.String(), "Token: new")
}
