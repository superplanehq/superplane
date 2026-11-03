package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestAddCheckSendsTheCheck(t *testing.T) {
	var body struct {
		Checks []map[string]any `json:"checks"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/factories/factory-1/orders/1823/checks", r.URL.Path)
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"checks":[{"key":"tests-pass","name":"Tests pass","score":9,"maxScore":10,"level":"LEVEL_POSITIVE","source":"agent"}]}`))
	}))
	t.Cleanup(server.Close)
	ctx, stdout := newTasksTestContext(t, server.URL)

	workspace, taskID := "factory-1", "1823"
	key, name, level, reason := "tests-pass", "Tests pass", "Positive", "All unit tests pass."
	score, maxScore := 9.0, 10.0
	err := (&checkAddCommand{
		workspace: &workspace,
		taskID:    &taskID,
		key:       &key,
		name:      &name,
		score:     &score,
		maxScore:  &maxScore,
		level:     &level,
		reason:    &reason,
	}).Execute(ctx)

	require.NoError(t, err)
	require.Len(t, body.Checks, 1)
	assert.Equal(t, "tests-pass", body.Checks[0]["key"])
	assert.Equal(t, "LEVEL_POSITIVE", body.Checks[0]["level"])
	assert.Equal(t, 9.0, body.Checks[0]["score"])
	assert.Equal(t, "All unit tests pass.", body.Checks[0]["reason"])
	assert.Contains(t, stdout.String(), "tests-pass")
	assert.Contains(t, stdout.String(), "9/10")
}

func TestCheckListUsesTheTaskFromTheEnvironment(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order":{"id":"order-1"}}`))
	}))
	t.Cleanup(server.Close)
	ctx, _ := newTasksTestContext(t, server.URL)
	t.Setenv(TaskIDEnv, "42")

	workspace, empty, explicit := "factory-1", "", "7"
	require.NoError(t, (&checkListCommand{workspace: &workspace, taskID: &empty}).Execute(ctx))
	require.NoError(t, (&checkListCommand{workspace: &workspace, taskID: &explicit}).Execute(ctx))

	assert.Equal(t, []string{
		"/api/v1/factories/factory-1/orders/42",
		"/api/v1/factories/factory-1/orders/7",
	}, paths)
}

func TestAddCheckRejectsAnUnknownLevel(t *testing.T) {
	level := "great"
	err := (&checkAddCommand{level: &level}).Execute(core.CommandContext{})
	require.EqualError(t, err, "--level must be positive, caution, or critical")
}

func TestAddFileArtifactUploadsTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "login.png")
	require.NoError(t, os.WriteFile(path, []byte("png-bytes"), 0o600))

	var (
		fileRequest     map[string]any
		uploaded        []byte
		uploadAuth      string
		uploadType      string
		artifactRequest struct {
			Type string         `json:"type"`
			Data map[string]any `json:"data"`
		}
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/factories/factory-1/orders/1823/files":
			require.NoError(t, json.Unmarshal(raw, &fileRequest))
			_, _ = w.Write([]byte(`{"file":{"id":"file-1"}}`))
		case r.Method == http.MethodPut && r.URL.Path == "/api/v1/files/file-1/content":
			uploaded = raw
			uploadAuth = r.Header.Get("Authorization")
			uploadType = r.Header.Get("Content-Type")
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/factories/factory-1/orders/1823/artifacts":
			require.NoError(t, json.Unmarshal(raw, &artifactRequest))
			_, _ = w.Write([]byte(`{"artifact":{"id":"artifact-1","type":"TYPE_FILE"}}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	ctx, _ := newTasksTestContext(t, server.URL)

	workspace, taskID, typ, title := "factory-1", "1823", "file", "Login screen"
	err := (&artifactAddCommand{workspace: &workspace, taskID: &taskID, typ: &typ, title: &title, file: &path}).Execute(ctx)

	require.NoError(t, err)
	assert.Equal(t, "artifact", fileRequest["purpose"])
	assert.Equal(t, "image/png", fileRequest["contentType"])
	assert.Equal(t, "login.png", fileRequest["filename"])
	assert.Equal(t, []byte("png-bytes"), uploaded)
	assert.Equal(t, "Bearer test-token", uploadAuth)
	assert.Equal(t, "image/png", uploadType)
	assert.Equal(t, "TYPE_FILE", artifactRequest.Type)
	assert.Equal(t, map[string]any{"fileId": "file-1", "title": "Login screen"}, artifactRequest.Data)
}

func TestAddFileArtifactRejectsAnUnsupportedFile(t *testing.T) {
	workspace, taskID, typ, path := "factory-1", "1823", "file", "notes.txt"
	err := (&artifactAddCommand{workspace: &workspace, taskID: &taskID, typ: &typ, file: &path}).Execute(core.CommandContext{})
	require.ErrorContains(t, err, "file type is not supported")
}

func TestGetArtifactPrintsTheMarkdownBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/factories/factory-1/orders/1823/artifacts", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"artifacts":[
			{"id":"a-1","type":"TYPE_LINK","data":{"url":"https://example.com"}},
			{"id":"a-2","key":"plan","type":"TYPE_MARKDOWN","data":{"title":"PLAN.md","body":"# Plan"}}
		]}`))
	}))
	t.Cleanup(server.Close)
	ctx, stdout := newTasksTestContext(t, server.URL)

	workspace, taskID, key := "factory-1", "1823", "plan"
	err := (&artifactGetCommand{workspace: &workspace, taskID: &taskID, key: &key}).Execute(ctx)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Title: PLAN.md")
	assert.Contains(t, stdout.String(), "# Plan")
	assert.NotContains(t, stdout.String(), "https://example.com")
}

func newTasksTestContext(t *testing.T, serverURL string) (core.CommandContext, *bytes.Buffer) {
	t.Helper()
	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer("text", stdout)
	require.NoError(t, err)
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{{URL: serverURL}}
	config.DefaultHeader["Authorization"] = "Bearer test-token"
	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	}, stdout
}
