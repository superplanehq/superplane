package executions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type fakeReplayConfig struct {
	activeApp string
}

func (f *fakeReplayConfig) GetActiveApp() string          { return f.activeApp }
func (f *fakeReplayConfig) SetActiveApp(string) error     { return nil }
func (f *fakeReplayConfig) GetActiveFactory() string      { return "" }
func (f *fakeReplayConfig) SetActiveFactory(string) error { return nil }
func (f *fakeReplayConfig) GetURL() string                { return "" }

func newReplayCommandContext(
	t *testing.T,
	server *httptest.Server,
	outputFormat string,
	activeApp string,
) (core.CommandContext, *bytes.Buffer) {
	t.Helper()

	stdout := bytes.NewBuffer(nil)
	renderer, err := core.NewRenderer(outputFormat, stdout)
	require.NoError(t, err)

	config := openapi_client.NewConfiguration()
	config.Servers = openapi_client.ServerConfigurations{{URL: server.URL}}

	cobraCmd := &cobra.Command{}
	cobraCmd.SetOut(stdout)

	return core.CommandContext{
		Context:  context.Background(),
		Cmd:      cobraCmd,
		API:      openapi_client.NewAPIClient(config),
		Renderer: renderer,
		Config:   &fakeReplayConfig{activeApp: activeApp},
	}, stdout
}

func fastPollTimes() (*time.Duration, *time.Duration) {
	timeout := 200 * time.Millisecond
	interval := 5 * time.Millisecond
	return &timeout, &interval
}

func TestReplayCommandCreatesReplayAndPollsExecution_Text(t *testing.T) {
	var listCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/replay":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"runId":"run-001","queueItemIds":["queue-001"]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/executions":
			listCalls++
			w.Header().Set("Content-Type", "application/json")
			if listCalls < 2 {
				_, _ = fmt.Fprint(w, `{"executions":[]}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"executions":[{"id":"exec-999","nodeId":"node-001","runId":"run-001","state":"STATE_FINISHED"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newReplayCommandContext(t, server, "text", "")
	canvasID := "canvas-001"
	nodeID := "node-001"
	executionID := "exec-001"
	timeout, interval := fastPollTimes()

	cmd := &ReplayCommand{
		CanvasID:     &canvasID,
		NodeID:       &nodeID,
		ExecutionID:  &executionID,
		PollTimeout:  timeout,
		PollInterval: interval,
	}

	require.NoError(t, cmd.Execute(ctx))
	require.GreaterOrEqual(t, listCalls, 2, "poll loop must retry, not just check once")

	raw := stdout.String()
	require.Contains(t, raw, "exec-999", "expected the polled execution id in text output, got: %s", raw)
	require.Contains(t, raw, "run-001")
}

func TestReplayCommandCreatesReplayAndPollsExecution_JSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/replay":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"runId":"run-002","queueItemIds":["queue-002"]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/executions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"executions":[{"id":"exec-777","nodeId":"node-001","runId":"run-002","state":"STATE_FINISHED"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newReplayCommandContext(t, server, "json", "")
	canvasID := "canvas-001"
	nodeID := "node-001"
	executionID := "exec-002"
	timeout, interval := fastPollTimes()

	cmd := &ReplayCommand{
		CanvasID:     &canvasID,
		NodeID:       &nodeID,
		ExecutionID:  &executionID,
		PollTimeout:  timeout,
		PollInterval: interval,
	}

	require.NoError(t, cmd.Execute(ctx))

	var output replayOutput
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &output))
	require.Equal(t, "run-002", output.RunID)
	require.Equal(t, "exec-777", output.ExecutionID, "expected discovered execution id in JSON output, got %q", output.ExecutionID)
	require.False(t, output.Queued)
}

func TestReplayCommandReportsQueuedOnPollTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/replay":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"runId":"run-003","queueItemIds":["queue-003"]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/executions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"executions":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newReplayCommandContext(t, server, "text", "")
	canvasID := "canvas-001"
	nodeID := "node-001"
	executionID := "exec-003"
	timeout := 30 * time.Millisecond
	interval := 5 * time.Millisecond

	cmd := &ReplayCommand{
		CanvasID:     &canvasID,
		NodeID:       &nodeID,
		ExecutionID:  &executionID,
		PollTimeout:  &timeout,
		PollInterval: &interval,
	}

	require.NoError(t, cmd.Execute(ctx), "a poll timeout must not be surfaced as a command error")

	raw := stdout.String()
	require.Contains(t, raw, "run-003")
	require.Contains(t, raw, "queued", "expected the output to say the replay is queued, got: %s", raw)
}

func TestReplayCommandPostsTheSourceExecutionItWasGiven(t *testing.T) {
	var requestBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/replay":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&requestBody))
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"runId":"run-007","queueItemIds":["queue-007"]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/executions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"executions":[{"id":"exec-007","nodeId":"node-001","runId":"run-007","state":"STATE_FINISHED"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	ctx, _ := newReplayCommandContext(t, server, "text", "")
	canvasID := "canvas-001"
	nodeID := "node-001"
	executionID := "exec-source-042"
	timeout, interval := fastPollTimes()

	cmd := &ReplayCommand{
		CanvasID:     &canvasID,
		NodeID:       &nodeID,
		ExecutionID:  &executionID,
		PollTimeout:  timeout,
		PollInterval: interval,
	}

	require.NoError(t, cmd.Execute(ctx))
	require.Equal(t, executionID, requestBody["sourceExecutionId"], "the replay must be requested from the source execution the caller named")
}

func TestReplayCommandSurfacesExecutionListFailureInsteadOfReportingQueued(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/replay":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"runId":"run-008","queueItemIds":["queue-008"]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/canvases/canvas-001/nodes/node-001/executions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprint(w, `{"code":16,"message":"unauthenticated"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	ctx, stdout := newReplayCommandContext(t, server, "text", "")
	canvasID := "canvas-001"
	nodeID := "node-001"
	executionID := "exec-008"
	timeout, interval := fastPollTimes()

	cmd := &ReplayCommand{
		CanvasID:     &canvasID,
		NodeID:       &nodeID,
		ExecutionID:  &executionID,
		PollTimeout:  timeout,
		PollInterval: interval,
	}

	err := cmd.Execute(ctx)
	require.Error(t, err, "a rejected executions request must not be reported as a busy node")
	require.Contains(t, err.Error(), "run-008", "the error must still name the run the replay created")
	require.NotContains(t, stdout.String(), "queued")
}

func TestReplayCommandSurfacesAPIErrorClearly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"code":3,"message":"trigger nodes cannot be replayed"}`)
	}))
	t.Cleanup(server.Close)

	ctx, _ := newReplayCommandContext(t, server, "text", "")
	canvasID := "canvas-001"
	nodeID := "trigger-node"
	executionID := "exec-004"
	timeout, interval := fastPollTimes()

	cmd := &ReplayCommand{
		CanvasID:     &canvasID,
		NodeID:       &nodeID,
		ExecutionID:  &executionID,
		PollTimeout:  timeout,
		PollInterval: interval,
	}

	// FormatCommandError is what core.Bind applies to every command's error in
	// the real CLI (cmd/cli/main.go's only path to a rendered message) -
	// wrap here the same way so this test proves what a real invocation
	// prints, not the SDK's raw unformatted status text.
	err := core.FormatCommandError(cmd.Execute(ctx))
	require.Error(t, err)
	require.Contains(t, err.Error(), "trigger nodes cannot be replayed")
}

func TestReplayCommandRequiresCanvas(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	ctx, _ := newReplayCommandContext(t, server, "text", "")
	empty := ""
	nodeID := "node-001"
	executionID := "exec-005"

	cmd := &ReplayCommand{
		CanvasID:    &empty,
		NodeID:      &nodeID,
		ExecutionID: &executionID,
	}

	err := cmd.Execute(ctx)
	require.Error(t, err, "expected a usage error when canvas is not provided, got nil")
}

func TestReplayCommandRequiresExecutionID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	ctx, _ := newReplayCommandContext(t, server, "text", "")
	canvasID := "canvas-001"
	nodeID := "node-001"
	empty := ""

	cmd := &ReplayCommand{
		CanvasID:    &canvasID,
		NodeID:      &nodeID,
		ExecutionID: &empty,
	}

	err := cmd.Execute(ctx)
	require.Error(t, err, "expected a usage error when source execution id is not provided, got nil")
	require.Contains(t, err.Error(), "execution")
}

func TestReplayCommandRequiresNodeID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)

	ctx, _ := newReplayCommandContext(t, server, "text", "")
	canvasID := "canvas-001"
	empty := ""
	executionID := "exec-006"

	cmd := &ReplayCommand{
		CanvasID:    &canvasID,
		NodeID:      &empty,
		ExecutionID: &executionID,
	}

	err := cmd.Execute(ctx)
	require.Error(t, err, "expected a usage error when node id is not provided, got nil")
	require.Contains(t, err.Error(), "node")
}
