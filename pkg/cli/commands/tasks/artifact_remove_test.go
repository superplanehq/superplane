package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestRemoveArtifactSendsTheKey(t *testing.T) {
	var body struct {
		Key   string `json:"key"`
		Title string `json:"title"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/factories/factory-1/orders/1823/artifacts/remove", r.URL.Path)
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"artifact-1","title":"PLAN.md"}`))
	}))
	t.Cleanup(server.Close)

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer("text", stdout)
	require.NoError(t, err)
	cmd := &cobra.Command{}
	cmd.SetOut(stdout)
	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{{URL: server.URL}}

	workspace := "factory-1"
	taskID := "1823"
	key := "plan"
	err = (&artifactRemoveCommand{workspace: &workspace, taskID: &taskID, key: &key}).Execute(core.CommandContext{
		Context:  context.Background(),
		Cmd:      cmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	})
	require.NoError(t, err)
	assert.Equal(t, "plan", body.Key)
	assert.Empty(t, body.Title)
	assert.Contains(t, stdout.String(), "Removed PLAN.md.")
}

func TestRemoveArtifactRequiresKeyOrTitle(t *testing.T) {
	workspace := "factory-1"
	taskID := "1823"
	err := (&artifactRemoveCommand{workspace: &workspace, taskID: &taskID}).Execute(core.CommandContext{})
	require.EqualError(t, err, "pass --key or --title")

	key := "plan"
	title := "PLAN.md"
	err = (&artifactRemoveCommand{workspace: &workspace, taskID: &taskID, key: &key, title: &title}).Execute(core.CommandContext{})
	require.EqualError(t, err, "pass --key or --title")
}
