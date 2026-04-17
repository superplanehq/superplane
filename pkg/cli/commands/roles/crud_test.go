package roles

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type rolePayload struct {
	DomainType string `json:"domainType"`
	DomainID   string `json:"domainId"`
	Role       struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Spec struct {
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
			Permissions []struct {
				Resource string `json:"resource"`
				Action   string `json:"action"`
			} `json:"permissions"`
		} `json:"spec"`
	} `json:"role"`
}

const roleFileYAML = `apiVersion: v1
kind: Role
metadata:
  name: release_manager
spec:
  displayName: Release Manager
  description: Can publish releases
  permissions:
    - resource: canvases
      action: read
    - resource: canvases
      action: publish
`

func writeRoleFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := dir + "/role.yaml"
	require.NoError(t, os.WriteFile(path, []byte(roleFileYAML), 0644))
	return path
}

func TestCreateRoleFromFile(t *testing.T) {
	var seen rolePayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/roles":
			body, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(body, &seen))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"role":{"metadata":{"name":"release_manager"},"spec":{"displayName":"Release Manager"}}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	path := writeRoleFile(t)
	ctx, stdout := newTestContext(t, server, "text")
	require.NoError(t, (&createCommand{file: &path}).Execute(ctx))

	require.Equal(t, "DOMAIN_TYPE_ORGANIZATION", seen.DomainType)
	require.Equal(t, testOrgID, seen.DomainID)
	require.Equal(t, "release_manager", seen.Role.Metadata.Name)
	require.Equal(t, "Release Manager", seen.Role.Spec.DisplayName)
	require.Len(t, seen.Role.Spec.Permissions, 2)
	require.Equal(t, "canvases", seen.Role.Spec.Permissions[0].Resource)
	require.Equal(t, "read", seen.Role.Spec.Permissions[0].Action)
	require.Contains(t, stdout.String(), "Name: release_manager")
}

func TestCreateRoleRequiresFile(t *testing.T) {
	ctx, _ := newTestContext(t, nil, "text")
	empty := ""
	err := (&createCommand{file: &empty}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--file is required")
}

func TestUpdateRoleFromFile(t *testing.T) {
	var seen rolePayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.URL.Path == "/api/v1/roles/release_manager":
			require.Equal(t, http.MethodPut, r.Method)
			body, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(body, &seen))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"role":{"metadata":{"name":"release_manager"},"spec":{"displayName":"Release Manager"}}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	path := writeRoleFile(t)
	ctx, _ := newTestContext(t, server, "text")
	require.NoError(t, (&updateCommand{file: &path}).Execute(ctx))
	require.Equal(t, "release_manager", seen.Role.Metadata.Name)
}

func TestUpdateRoleRejectsPositional(t *testing.T) {
	ctx, _ := newTestContext(t, nil, "text")
	path := "x"
	ctx.Args = []string{"release_manager"}
	err := (&updateCommand{file: &path}).Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not accept positional")
}

func TestDeleteRole(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/roles/release_manager":
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
	ctx.Args = []string{"release_manager"}
	require.NoError(t, (&deleteCommand{}).Execute(ctx))
	require.True(t, deleted)
	require.Contains(t, stdout.String(), "Role deleted: release_manager")
}
