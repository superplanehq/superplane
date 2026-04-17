package groups

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
)

type updateBody struct {
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

func newUpdateServer(t *testing.T, seen *updateBody) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/me":
			writeMeResponse(w)
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/groups/engineers":
			payload, _ := io.ReadAll(r.Body)
			require.NoError(t, json.Unmarshal(payload, seen))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"group":{"metadata":{"name":"engineers"},"spec":{"displayName":"Engineers"}}}`))
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// newUpdateContext builds a CommandContext whose cobra.Command has the update
// flags registered, so tests can emulate which flags the user "changed".
func newUpdateContext(t *testing.T, server *httptest.Server, changed map[string]string) (updateCommand, *testContextResult) {
	t.Helper()
	ctx, stdout := newTestContext(t, server, "text")

	displayName := ""
	description := ""
	role := ""
	filePath := ""

	cobraCmd := ctx.Cmd
	cobraCmd.Flags().StringVar(&displayName, "display-name", "", "")
	cobraCmd.Flags().StringVar(&description, "description", "", "")
	cobraCmd.Flags().StringVar(&role, "role", "", "")
	cobraCmd.Flags().StringVarP(&filePath, "file", "f", "", "")

	for name, value := range changed {
		require.NoError(t, cobraCmd.Flags().Set(name, value))
	}

	cmd := updateCommand{
		file:        &filePath,
		displayName: &displayName,
		description: &description,
		role:        &role,
	}

	return cmd, &testContextResult{ctx: ctx, stdout: stdout, cobraCmd: cobraCmd}
}

type testContextResult struct {
	ctx      core.CommandContext
	stdout   interface{ String() string }
	cobraCmd *cobra.Command
}

func TestUpdateInlineSendsOnlyChangedFields(t *testing.T) {
	var seen updateBody
	server := newUpdateServer(t, &seen)

	cmd, got := newUpdateContext(t, server, map[string]string{
		"display-name": "Engineers",
	})
	got.ctx.Args = []string{"engineers"}

	require.NoError(t, cmd.Execute(got.ctx))
	require.Equal(t, "DOMAIN_TYPE_ORGANIZATION", seen.DomainType)
	require.Equal(t, testOrgID, seen.DomainID)
	require.Equal(t, "engineers", seen.Group.Metadata.Name)
	require.Equal(t, "Engineers", seen.Group.Spec.DisplayName)
	// Unchanged fields must be omitted so the backend merge preserves them.
	require.Empty(t, seen.Group.Spec.Description)
	require.Empty(t, seen.Group.Spec.Role)
}

func TestUpdateInlineSendsAllThreeWhenProvided(t *testing.T) {
	var seen updateBody
	server := newUpdateServer(t, &seen)

	cmd, got := newUpdateContext(t, server, map[string]string{
		"display-name": "Engineers",
		"description":  "Engineering team",
		"role":         "org_admin",
	})
	got.ctx.Args = []string{"engineers"}

	require.NoError(t, cmd.Execute(got.ctx))
	require.Equal(t, "Engineers", seen.Group.Spec.DisplayName)
	require.Equal(t, "Engineering team", seen.Group.Spec.Description)
	require.Equal(t, "org_admin", seen.Group.Spec.Role)
}

func TestUpdateFromFile(t *testing.T) {
	var seen updateBody
	server := newUpdateServer(t, &seen)

	dir := t.TempDir()
	path := dir + "/group.yaml"
	content := []byte("apiVersion: v1\nkind: Group\nmetadata:\n  name: engineers\nspec:\n  displayName: FromFile\n  role: org_viewer\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	cmd, got := newUpdateContext(t, server, map[string]string{
		"file": path,
	})

	require.NoError(t, cmd.Execute(got.ctx))
	require.Equal(t, "engineers", seen.Group.Metadata.Name)
	require.Equal(t, "FromFile", seen.Group.Spec.DisplayName)
	require.Equal(t, "org_viewer", seen.Group.Spec.Role)
}

func TestUpdateFromFileRequiresMetadataName(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/group.yaml"
	content := []byte("apiVersion: v1\nkind: Group\nspec:\n  displayName: No Name\n")
	require.NoError(t, os.WriteFile(path, content, 0644))

	cmd, got := newUpdateContext(t, newMeOnlyServer(t), map[string]string{
		"file": path,
	})

	err := cmd.Execute(got.ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "metadata.name is required")
}

func TestUpdateRejectsPositionalWithFile(t *testing.T) {
	cmd, got := newUpdateContext(t, newMeOnlyServer(t), map[string]string{
		"file": "/nonexistent",
	})
	got.ctx.Args = []string{"engineers"}

	err := cmd.Execute(got.ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot combine a positional name with --file")
}

func TestUpdateRequiresPositionalOrFile(t *testing.T) {
	cmd, got := newUpdateContext(t, newMeOnlyServer(t), map[string]string{})

	err := cmd.Execute(got.ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "group name positional argument is required")
}

func TestUpdateRequiresAtLeastOneFlag(t *testing.T) {
	cmd, got := newUpdateContext(t, newMeOnlyServer(t), map[string]string{})
	got.ctx.Args = []string{"engineers"}

	err := cmd.Execute(got.ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one of --display-name, --description, --role, or --file")
}
