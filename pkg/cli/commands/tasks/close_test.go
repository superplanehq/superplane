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

func TestCloseTaskSendsTheResult(t *testing.T) {
	var body struct {
		Result string `json:"result"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method)
		require.Equal(t, "/api/v1/factories/factory-1/orders/1823/close", r.URL.Path)
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order":{"key":"SUPER-1823","result":"RESULT_REJECTED"}}`))
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
	result := "rejected"
	err = (&taskCloseCommand{workspace: &workspace, taskID: &taskID, result: &result}).Execute(core.CommandContext{
		Context:  context.Background(),
		Cmd:      cmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	})
	require.NoError(t, err)
	assert.Equal(t, "RESULT_REJECTED", body.Result)
	assert.Contains(t, stdout.String(), "Closed SUPER-1823")
	assert.Contains(t, stdout.String(), "Rejected")
}

func TestCloseTaskRequiresTask(t *testing.T) {
	workspace := "factory-1"
	err := (&taskCloseCommand{workspace: &workspace}).Execute(core.CommandContext{})
	require.EqualError(t, err, "--task is required")
}

func TestCloseTaskRejectsAnUnknownResult(t *testing.T) {
	workspace := "factory-1"
	taskID := "1823"
	result := "archived"
	err := (&taskCloseCommand{workspace: &workspace, taskID: &taskID, result: &result}).Execute(core.CommandContext{})
	require.EqualError(t, err, "--result must be completed, rejected, or failed")
}
