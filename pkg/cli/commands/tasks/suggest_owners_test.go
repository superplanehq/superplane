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

func TestSuggestOwnersPrintsRankedOwners(t *testing.T) {
	var body struct {
		Paths []string `json:"paths"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/factories/factory-1/orders/1823/suggest-owners", r.URL.Path)
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"paths": ["pkg/payments"],
			"currentOwners": [{"userId": "u1", "name": "Ada Lovelace", "githubLogin": "ada"}],
			"codeOwners": [{"userId": "u1", "name": "Ada Lovelace", "githubLogin": "ada", "commits": 3, "openTasks": 1}],
			"members": [{"userId": "u1", "name": "Ada Lovelace", "email": "ada@example.com", "openTasks": 1}],
			"historyNote": "SuperPlane could not read the code history. Use the members list."
		}`))
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
	paths := []string{" pkg/payments ", ""}
	err = (&taskSuggestOwnersCommand{workspace: &workspace, taskID: &taskID, paths: &paths}).Execute(core.CommandContext{
		Context:  context.Background(),
		Cmd:      cmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"pkg/payments"}, body.Paths)
	assert.Contains(t, stdout.String(), "Current owners:")
	assert.Contains(t, stdout.String(), "Ada Lovelace (ada)")
	assert.Contains(t, stdout.String(), "3 commits")
	assert.Contains(t, stdout.String(), "Ada Lovelace <ada@example.com>")
	assert.Contains(t, stdout.String(), "SuperPlane could not read the code history. Use the members list.")
}

func TestSuggestOwnersRequiresTask(t *testing.T) {
	workspace := "factory-1"
	err := (&taskSuggestOwnersCommand{workspace: &workspace}).Execute(core.CommandContext{})
	require.EqualError(t, err, "--task is required")
}
