package staging

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/yaml"
	"github.com/superplanehq/superplane/test/support/cli"
)

const testAppID = "4e9ae08d-0363-40d2-ba2c-5f6389a418d8"

func stagingPath(canvasID string) string {
	return "/api/v1/canvases/" + canvasID + "/staging"
}

func TestStatusCommandPrintsStagedPaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == stagingPath(testAppID) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":true,"stagedPaths":["canvas.yaml","console.yaml"],"baseVersionId":"version-1","stale":false}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{testAppID}

	err := (&statusCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, "M canvas.yaml\nM console.yaml\n", stdout.String())
}

func TestStatusCommandPrintsNothingWhenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == stagingPath(testAppID) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":false}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	ctx, stdout := cli.NewCommandContext(t, server, "text")
	ctx.Args = []string{testAppID}

	err := (&statusCommand{}).Execute(ctx)
	require.NoError(t, err)
	require.Empty(t, stdout.String())
}

func TestUpdateCommandStagesFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == stagingPath(testAppID) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":true,"stagedPaths":["canvas.yaml","README.md"]}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	canvasPath := filepath.Join(dir, "canvas.yaml")
	readmePath := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(canvasPath, []byte(
		"apiVersion: v1\nkind: Canvas\nmetadata:\n  id: "+testAppID+"\n  name: demo\nspec:\n  nodes: []\n  edges: []\n",
	), 0o644))
	require.NoError(t, os.WriteFile(readmePath, []byte("hello"), 0o644))

	files := []string{canvasPath, readmePath}
	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: testAppID})

	err := (&updateCommand{files: &files}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "canvas.yaml")
	require.Contains(t, stdout.String(), "README.md")
}

// TestUpdateCommand_ConsoleYAMLValidation locks in the "lenient parse
// plus shape check" pre-flight applied by `apps staging update`. The
// command must accept a grandfathered over-cap console (rejected by the
// strict parser it used to call) so users with an existing over-cap
// document can still re-stage it — including to reduce it — while
// still rejecting shape errors such as the wrong `kind`. Cap
// enforcement lives at commit time via `ValidateConsolePagesDelta`,
// not in this pre-flight.
func TestUpdateCommand_ConsoleYAMLValidation(t *testing.T) {
	overCapConsoleYAML := func() string {
		var b strings.Builder
		b.WriteString("apiVersion: v1\nkind: Console\nmetadata: {}\nspec:\n  panels:\n")
		for i := 0; i < yaml.MaxConsolePanelsPerPage+3; i++ {
			fmt.Fprintf(&b, "    - id: panel-%d\n      type: markdown\n      content: {}\n", i)
		}
		b.WriteString("  layout: []\n")
		return b.String()
	}

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "accepts grandfathered over-cap console",
			content: overCapConsoleYAML(),
		},
		{
			name:    "rejects wrong kind",
			content: "apiVersion: v1\nkind: Canvas\nmetadata: {}\nspec: {panels: [], layout: []}\n",
			wantErr: "unsupported kind",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPut && r.URL.Path == stagingPath(testAppID) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"stagingSummary":{"hasStaging":true,"stagedPaths":["console.yaml"]}}`))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			t.Cleanup(server.Close)

			dir := t.TempDir()
			consolePath := filepath.Join(dir, "console.yaml")
			require.NoError(t, os.WriteFile(consolePath, []byte(tc.content), 0o644))

			files := []string{consolePath}
			ctx, _ := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: testAppID})

			err := (&updateCommand{files: &files}).Execute(ctx)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestCommitCommandPrintsVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == stagingPath(testAppID)+"/commit" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":{"metadata":{"id":"version-2","canvasId":"` + testAppID + `"}}}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	message := "Ship it"
	ctx, stdout := cli.NewCommandContextWithConfig(t, server, "text", &cli.FakeConfig{ActiveApp: testAppID})

	err := (&commitCommand{message: &message}).Execute(ctx)
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Version: version-2")
}
