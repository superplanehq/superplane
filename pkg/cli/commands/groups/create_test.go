package groups

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type createBody struct {
	DomainType string `json:"domainType"`
	DomainID   string `json:"domainId"`
	Group      struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Spec struct {
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
			Role        string `json:"role"`
		} `json:"spec"`
	} `json:"group"`
}

func newCreateServer(t *testing.T, seen *createBody) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/groups":
			payload, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(payload, seen))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"group":{"metadata":{"name":"` + seen.Group.Metadata.Name + `"},"spec":{"displayName":"` + seen.Group.Spec.DisplayName + `"}}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestCreateInlinePassesFlags(t *testing.T) {
	var seen createBody
	server := newCreateServer(t, &seen)
	ctx, stdout := newTestContext(t, server, "text")
	ctx.Args = []string{"engineers"}

	display := "Engineers"
	description := "Engineering team"
	role := "org_admin"
	empty := ""
	cmd := &createCommand{
		file:        &empty,
		displayName: &display,
		description: &description,
		role:        &role,
	}
	require.NoError(t, cmd.Execute(ctx))

	require.Equal(t, "DOMAIN_TYPE_ORGANIZATION", seen.DomainType)
	require.Equal(t, testOrgID, seen.DomainID)
	require.Equal(t, "engineers", seen.Group.Metadata.Name)
	require.Equal(t, "Engineers", seen.Group.Spec.DisplayName)
	require.Equal(t, "Engineering team", seen.Group.Spec.Description)
	require.Equal(t, "org_admin", seen.Group.Spec.Role)
	require.Contains(t, stdout.String(), "Name: engineers")
}

func TestCreateFromFileOverridesPositional(t *testing.T) {
	var seen createBody
	server := newCreateServer(t, &seen)
	ctx, _ := newTestContext(t, server, "text")

	dir := t.TempDir()
	path := dir + "/group.yaml"
	content := []byte("apiVersion: v1\nkind: Group\nmetadata:\n  name: from-file\nspec:\n  displayName: FromFile\n  role: org_viewer\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	displayEmpty := ""
	descriptionEmpty := ""
	roleEmpty := ""
	cmd := &createCommand{
		file:        &path,
		displayName: &displayEmpty,
		description: &descriptionEmpty,
		role:        &roleEmpty,
	}
	require.NoError(t, cmd.Execute(ctx))
	require.Equal(t, "from-file", seen.Group.Metadata.Name)
	require.Equal(t, "FromFile", seen.Group.Spec.DisplayName)
}

func newMeOnlyServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/me" {
			writeMeResponse(w)
			return
		}
		t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestCreateRejectsPositionalWithFile(t *testing.T) {
	ctx, _ := newTestContext(t, newMeOnlyServer(t), "text")
	ctx.Args = []string{"engineers"}

	filePath := "/nonexistent"
	empty := ""
	cmd := &createCommand{
		file:        &filePath,
		displayName: &empty,
		description: &empty,
		role:        &empty,
	}
	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot combine a positional name with --file")
}

func TestCreateRequiresNameOrFile(t *testing.T) {
	ctx, _ := newTestContext(t, newMeOnlyServer(t), "text")
	empty := ""
	cmd := &createCommand{
		file:        &empty,
		displayName: &empty,
		description: &empty,
		role:        &empty,
	}
	err := cmd.Execute(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "group name positional argument or --file is required")
}
