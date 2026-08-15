package executions

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// defaultReplayPollTimeout/Interval bound how long the CLI waits for
// NodeQueueWorker to turn the replay's queue item into an execution before
// reporting it as queued instead of failing. Replay enters through the node
// queue, so no execution exists at response time - a busy node
// legitimately defers the replay rather than erroring.
const (
	defaultReplayPollTimeout  = 30 * time.Second
	defaultReplayPollInterval = 500 * time.Millisecond

	// The server caps a page at 25 (MaxLimit), so asking for more only
	// overstates what a poll actually inspects.
	replayPollListLimit = 25
)

type ReplayCommand struct {
	CanvasID     *string
	NodeID       *string
	ExecutionID  *string
	PollTimeout  *time.Duration
	PollInterval *time.Duration
}

type replayOutput struct {
	CanvasID          string   `json:"canvas_id"`
	NodeID            string   `json:"node_id"`
	SourceExecutionID string   `json:"source_execution_id"`
	RunID             string   `json:"run_id"`
	QueueItemIDs      []string `json:"queue_item_ids,omitempty"`
	ExecutionID       string   `json:"execution_id,omitempty"`
	Queued            bool     `json:"queued,omitempty"`
}

func (c *ReplayCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveAppID(ctx, *c.CanvasID)
	if err != nil {
		return err
	}

	nodeID := strings.TrimSpace(*c.NodeID)
	if nodeID == "" {
		return fmt.Errorf("node id is required; pass --node-id")
	}

	sourceExecutionID := strings.TrimSpace(*c.ExecutionID)
	if sourceExecutionID == "" {
		return fmt.Errorf("source execution id is required; pass --execution-id")
	}

	body := openapi_client.NewCanvasesReplayNodeBody()
	body.SetSourceExecutionId(sourceExecutionID)

	response, _, err := ctx.API.CanvasNodeAPI.
		CanvasesReplayNode(ctx.Context, canvasID, nodeID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	output := replayOutput{
		CanvasID:          canvasID,
		NodeID:            nodeID,
		SourceExecutionID: sourceExecutionID,
		RunID:             response.GetRunId(),
		QueueItemIDs:      response.GetQueueItemIds(),
	}

	executionID, found, err := c.pollForExecution(ctx, canvasID, nodeID, output.RunID)
	if err != nil {
		return fmt.Errorf("replay created (run %s), but listing its executions failed: %w", output.RunID, err)
	}

	if found {
		output.ExecutionID = executionID
	} else {
		output.Queued = true
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(output)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderReplayText(stdout, output)
	})
}

// The last API error is carried out of the loop: a 401 or an unreachable
// server must not be reported as "not created yet".
func (c *ReplayCommand) pollForExecution(ctx core.CommandContext, canvasID, nodeID, runID string) (string, bool, error) {
	timeout := defaultReplayPollTimeout
	if c.PollTimeout != nil && *c.PollTimeout > 0 {
		timeout = *c.PollTimeout
	}

	interval := defaultReplayPollInterval
	if c.PollInterval != nil && *c.PollInterval > 0 {
		interval = *c.PollInterval
	}

	deadline := time.Now().Add(timeout)
	for {
		executionID, found, err := findExecutionForRun(ctx, canvasID, nodeID, runID)
		if found {
			return executionID, true, nil
		}

		if !time.Now().Before(deadline) {
			return "", false, err
		}

		select {
		case <-ctx.Context.Done():
			return "", false, err
		case <-time.After(interval):
		}
	}
}

func findExecutionForRun(ctx core.CommandContext, canvasID, nodeID, runID string) (string, bool, error) {
	response, _, err := ctx.API.CanvasNodeAPI.
		CanvasesListNodeExecutions(ctx.Context, canvasID, nodeID).
		Limit(replayPollListLimit).
		Execute()
	if err != nil {
		return "", false, err
	}

	for _, execution := range response.GetExecutions() {
		if execution.GetRunId() == runID {
			return execution.GetId(), true, nil
		}
	}

	return "", false, nil
}

func renderReplayText(stdout io.Writer, output replayOutput) error {
	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintf(writer, "Canvas\t%s\n", output.CanvasID)
	_, _ = fmt.Fprintf(writer, "Node\t%s\n", output.NodeID)
	_, _ = fmt.Fprintf(writer, "Source execution\t%s\n", output.SourceExecutionID)
	_, _ = fmt.Fprintf(writer, "Run\t%s\n", output.RunID)
	if output.ExecutionID != "" {
		_, _ = fmt.Fprintf(writer, "Execution\t%s\n", output.ExecutionID)
	}
	if err := writer.Flush(); err != nil {
		return err
	}

	if output.ExecutionID != "" {
		_, err := fmt.Fprintf(stdout, "Replay created: execution %s (run %s)\n", output.ExecutionID, output.RunID)
		return err
	}

	_, err := fmt.Fprintf(
		stdout,
		"Replay queued: run %s (node is busy; execution not created yet - check again with \"superplane executions list --node-id %s\")\n",
		output.RunID,
		output.NodeID,
	)
	return err
}
