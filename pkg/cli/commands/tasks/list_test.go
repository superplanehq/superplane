package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func TestTaskListTruncatedMessage(t *testing.T) {
	message := taskListTruncatedMessage()
	assert.Contains(t, message, "first 100 matching tasks")
	assert.Contains(t, message, "--user, --state, --result, or --unassigned")
	assert.Contains(t, message, "--all")
}

func TestTaskList_DefaultStopsAfterFirstPage(t *testing.T) {
	server := newTaskListServer(t)
	ctx, stdout, stderr := newTaskListCommandContext(t, server, "text")
	workspace := "factory-1"
	cmd := &taskListCommand{workspace: &workspace}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, server.requestCount)
	assert.Contains(t, stdout.String(), "1")
	assert.Contains(t, stdout.String(), "First")
	assert.NotContains(t, stdout.String(), "Second")
	assert.Contains(t, stdout.String(), taskListTruncatedMessage())
	assert.Empty(t, stderr.String())
}

func TestTaskList_AllWalksEveryPage(t *testing.T) {
	server := newTaskListServer(t)
	ctx, stdout, stderr := newTaskListCommandContext(t, server, "text")
	workspace := "factory-1"
	all := true
	cmd := &taskListCommand{workspace: &workspace, all: &all}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	assert.Equal(t, 2, server.requestCount)
	assert.Equal(t, "order-1", server.lastBeforeID)
	assert.Contains(t, stdout.String(), "First")
	assert.Contains(t, stdout.String(), "Second")
	assert.NotContains(t, stdout.String(), taskListTruncatedMessage())
	assert.Empty(t, stderr.String())
}

func TestTaskList_JSONReturnsFirstPageOnly(t *testing.T) {
	server := newTaskListServer(t)
	ctx, stdout, stderr := newTaskListCommandContext(t, server, "json")
	workspace := "factory-1"
	cmd := &taskListCommand{workspace: &workspace}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, server.requestCount)
	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 1)
	assert.Equal(t, "order-1", result[0]["id"])
	assert.Empty(t, stderr.String())
	assert.NotContains(t, stdout.String(), taskListTruncatedMessage())
}

func TestTaskList_JSONAllReturnsEveryPage(t *testing.T) {
	server := newTaskListServer(t)
	ctx, stdout, stderr := newTaskListCommandContext(t, server, "json")
	workspace := "factory-1"
	all := true
	cmd := &taskListCommand{workspace: &workspace, all: &all}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	var result []map[string]any
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	require.Len(t, result, 2)
	assert.Equal(t, "order-1", result[0]["id"])
	assert.Equal(t, "order-2", result[1]["id"])
	assert.Empty(t, stderr.String())
}

func TestTaskList_CompleteFirstPageHasNoHint(t *testing.T) {
	server := newTaskListServer(t)
	server.complete = true
	ctx, stdout, _ := newTaskListCommandContext(t, server, "text")
	workspace := "factory-1"
	cmd := &taskListCommand{workspace: &workspace}

	err := cmd.Execute(ctx)
	require.NoError(t, err)

	assert.Equal(t, 1, server.requestCount)
	assert.Contains(t, stdout.String(), "First")
	assert.NotContains(t, stdout.String(), taskListTruncatedMessage())
}

type taskListServer struct {
	*httptest.Server
	requestCount int
	lastBeforeID string
	complete     bool
}

func newTaskListServer(t *testing.T) *taskListServer {
	t.Helper()
	server := &taskListServer{}
	server.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/factories/factory-1/orders", r.URL.Path)
		require.Equal(t, "100", r.URL.Query().Get("limit"))
		require.Equal(t, "STATE_OPEN", r.URL.Query().Get("states"))

		server.requestCount++
		server.lastBeforeID = r.URL.Query().Get("beforeId")
		w.Header().Set("Content-Type", "application/json")

		if server.complete {
			_, _ = fmt.Fprint(w, taskListPageJSON("order-1", "1", "First", false))
			return
		}

		if server.lastBeforeID == "" {
			_, _ = fmt.Fprint(w, taskListPageJSON("order-1", "1", "First", true))
			return
		}

		require.Equal(t, "order-1", server.lastBeforeID)
		_, _ = fmt.Fprint(w, taskListPageJSON("order-2", "2", "Second", false))
	}))
	t.Cleanup(server.Close)
	return server
}

func taskListPageJSON(id, number, title string, hasNextPage bool) string {
	return fmt.Sprintf(`{
		"orders": [
			{
				"id": %q,
				"number": %q,
				"title": %q,
				"state": "STATE_OPEN",
				"result": "RESULT_UNSPECIFIED",
				"createdAt": "2025-01-15T10:00:00Z"
			}
		],
		"hasNextPage": %t
	}`, id, number, title, hasNextPage)
}

func newTaskListCommandContext(
	t *testing.T,
	server *taskListServer,
	outputFormat string,
) (core.CommandContext, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)
	cobraCmd.SetErr(stderr)

	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{
		{URL: server.URL},
	}

	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
	}, stdout, stderr
}
