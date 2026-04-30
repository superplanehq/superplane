package groups

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type membersMutationBody struct {
	DomainType string `json:"domainType"`
	DomainID   string `json:"domainId"`
	UserID     string `json:"userId"`
	UserEmail  string `json:"userEmail"`
}

const listGroupUsersPayload = `{
  "users": [
    {"metadata":{"id":"u-1","email":"alice@example.com"},"spec":{"displayName":"Alice"}}
  ]
}`

func TestMembersList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/groups/engineers/users":
			require.Equal(t, "DOMAIN_TYPE_ORGANIZATION", r.URL.Query().Get("domainType"))
			require.Equal(t, testOrgID, r.URL.Query().Get("domainId"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(listGroupUsersPayload))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	ctx.Args = []string{"engineers"}
	require.NoError(t, (&membersListCommand{}).Execute(ctx))
	require.Contains(t, stdout.String(), "alice@example.com")
}

func TestMembersAddSendsUserID(t *testing.T) {
	var body membersMutationBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/groups/engineers/users":
			payload, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(payload, &body))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newTestContext(t, server, "text")
	ctx.Args = []string{"engineers", "u-1"}
	email := ""
	require.NoError(t, (&membersAddCommand{email: &email}).Execute(ctx))
	require.Equal(t, "u-1", body.UserID)
	require.Empty(t, body.UserEmail)
	require.Equal(t, testOrgID, body.DomainID)
	require.Contains(t, stdout.String(), "Added u-1 to group engineers")
}

func TestMembersAddSendsUserEmailFromPositional(t *testing.T) {
	var body membersMutationBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.URL.Path == "/api/v1/groups/engineers/users":
			payload, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(payload, &body))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := newTestContext(t, server, "text")
	ctx.Args = []string{"engineers", "eve@example.com"}
	email := ""
	require.NoError(t, (&membersAddCommand{email: &email}).Execute(ctx))
	require.Empty(t, body.UserID)
	require.Equal(t, "eve@example.com", body.UserEmail)
}

func TestMembersAddSendsUserEmail(t *testing.T) {
	var body membersMutationBody
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.URL.Path == "/api/v1/groups/engineers/users":
			payload, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(payload, &body))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := newTestContext(t, server, "text")
	ctx.Args = []string{"engineers"}
	email := "new@example.com"
	require.NoError(t, (&membersAddCommand{email: &email}).Execute(ctx))
	require.Empty(t, body.UserID)
	require.Equal(t, "new@example.com", body.UserEmail)
}

func TestMembersAddRequiresIdentifier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me" {
			writeMeResponse(w)
			return
		}
		t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(server.Close)

	ctx, _ := newTestContext(t, server, "text")
	ctx.Args = []string{"engineers"}
	empty := ""
	err := (&membersAddCommand{email: &empty}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "user id positional argument or --email")
}

func TestMembersRemoveUsesPatch(t *testing.T) {
	var body membersMutationBody
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.URL.Path == "/api/v1/groups/engineers/users/remove":
			require.Equal(t, http.MethodPatch, r.Method)
			called = true
			payload, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(payload, &body))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := newTestContext(t, server, "text")
	ctx.Args = []string{"engineers", "u-1"}
	empty := ""
	require.NoError(t, (&membersRemoveCommand{email: &empty}).Execute(ctx))
	require.True(t, called)
	require.Equal(t, "u-1", body.UserID)
}
