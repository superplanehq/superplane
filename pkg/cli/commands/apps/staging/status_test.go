package staging

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
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
