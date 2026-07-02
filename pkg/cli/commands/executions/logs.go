package executions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	runneraction "github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type LogsCommand struct {
	CanvasID    *string
	ExecutionID *string
	RunID       *string
	NodeID      *string
	Limit       *int64
}

type runnerLogTarget struct {
	ExecutionID string
	NodeID      string
}

type runnerLogOutput struct {
	CanvasID    string                       `json:"canvas_id"`
	ExecutionID string                       `json:"execution_id"`
	NodeID      string                       `json:"node_id,omitempty"`
	Count       int                          `json:"count"`
	Truncated   bool                         `json:"truncated,omitempty"`
	Records     []runneraction.LiveLogRecord `json:"records,omitempty"`
	Error       string                       `json:"error,omitempty"`
}

func (c *LogsCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveAppID(ctx, *c.CanvasID)
	if err != nil {
		return err
	}

	targets, err := c.resolveTargets(ctx, canvasID)
	if err != nil {
		return err
	}

	outputs := make([]runnerLogOutput, 0, len(targets))
	for _, target := range targets {
		outputs = append(outputs, c.fetchTargetLogs(ctx, canvasID, target))
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(outputs)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderRunnerLogsText(stdout, outputs)
	})
}

func (c *LogsCommand) resolveTargets(ctx core.CommandContext, canvasID string) ([]runnerLogTarget, error) {
	if strings.TrimSpace(*c.ExecutionID) != "" {
		return []runnerLogTarget{{ExecutionID: strings.TrimSpace(*c.ExecutionID)}}, nil
	}

	if strings.TrimSpace(*c.RunID) != "" {
		return c.resolveRunTargets(ctx, canvasID)
	}

	if strings.TrimSpace(*c.NodeID) != "" {
		return c.resolveLatestNodeTarget(ctx, canvasID)
	}

	return nil, fmt.Errorf("one of --execution-id, --run-id, or --node-id is required")
}

func (c *LogsCommand) resolveRunTargets(ctx core.CommandContext, canvasID string) ([]runnerLogTarget, error) {
	response, _, err := ctx.API.CanvasRunAPI.
		CanvasesDescribeRun(ctx.Context, canvasID, strings.TrimSpace(*c.RunID)).
		Execute()
	if err != nil {
		return nil, err
	}

	run, ok := response.GetRunOk()
	if !ok || run == nil {
		return nil, fmt.Errorf("run %q not found", strings.TrimSpace(*c.RunID))
	}

	executions := run.GetExecutions()
	runnerNodes, err := c.runnerNodeIDs(ctx, canvasID)
	if err != nil {
		return nil, err
	}

	targets := make([]runnerLogTarget, 0, len(executions))
	for _, execution := range executions {
		nodeID := execution.GetNodeId()
		if strings.TrimSpace(*c.NodeID) != "" && nodeID != strings.TrimSpace(*c.NodeID) {
			continue
		}
		if !runnerNodes[nodeID] {
			continue
		}
		targets = append(targets, runnerLogTarget{
			ExecutionID: execution.GetId(),
			NodeID:      nodeID,
		})
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no runner executions found for log target")
	}
	return targets, nil
}

func (c *LogsCommand) runnerNodeIDs(ctx core.CommandContext, canvasID string) (map[string]bool, error) {
	response, _, err := ctx.API.CanvasAPI.
		CanvasesDescribeCanvas(ctx.Context, canvasID).
		Execute()
	if err != nil {
		return nil, err
	}

	canvas, ok := response.GetCanvasOk()
	if !ok || canvas == nil {
		return nil, fmt.Errorf("canvas %q not found", canvasID)
	}

	spec := canvas.GetSpec()
	nodes := spec.GetNodes()
	ids := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		id := strings.TrimSpace(node.GetId())
		if id == "" || !runneraction.IsRunnerComponent(node.GetComponent()) {
			continue
		}
		ids[id] = true
	}
	return ids, nil
}

func (c *LogsCommand) resolveLatestNodeTarget(ctx core.CommandContext, canvasID string) ([]runnerLogTarget, error) {
	response, _, err := ctx.API.CanvasNodeAPI.
		CanvasesListNodeExecutions(ctx.Context, canvasID, strings.TrimSpace(*c.NodeID)).
		Limit(1).
		Execute()
	if err != nil {
		return nil, err
	}

	executions := response.GetExecutions()
	if len(executions) == 0 {
		return nil, fmt.Errorf("no executions found for node %q", strings.TrimSpace(*c.NodeID))
	}

	execution := executions[0]
	return []runnerLogTarget{{
		ExecutionID: execution.GetId(),
		NodeID:      execution.GetNodeId(),
	}}, nil
}

func (c *LogsCommand) fetchTargetLogs(ctx core.CommandContext, canvasID string, target runnerLogTarget) runnerLogOutput {
	output := runnerLogOutput{
		CanvasID:    canvasID,
		ExecutionID: target.ExecutionID,
		NodeID:      target.NodeID,
	}

	records, err := c.fetchExecutionLogs(ctx, canvasID, target.ExecutionID)
	if err != nil {
		output.Error = err.Error()
		return output
	}

	output.Count = len(records.Records)
	output.Truncated = records.Truncated
	output.Records = records.Records
	return output
}

func (c *LogsCommand) fetchExecutionLogs(ctx core.CommandContext, canvasID, executionID string) (*runneraction.LiveLogFetchResult, error) {
	session, err := c.fetchLiveLogSession(ctx, canvasID, executionID)
	if err != nil {
		return nil, err
	}

	records, err := runneraction.FetchLiveLogSessionRecords(ctx.Context, *session, runneraction.LiveLogFetchOptions{
		Limit:      normalizedLogLimit(c.Limit),
		HTTPClient: logHTTPClient(ctx.API.GetConfig()),
	})
	if err != nil {
		return nil, fmt.Errorf("fetch runner logs for execution %s: %w", executionID, err)
	}
	return records, nil
}

func (c *LogsCommand) fetchLiveLogSession(ctx core.CommandContext, canvasID, executionID string) (*runneraction.LiveLogSession, error) {
	config := ctx.API.GetConfig()
	if config == nil {
		return nil, fmt.Errorf("api client config is required")
	}

	baseURL, err := config.ServerURLWithContext(ctx.Context, "")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("api_url is required")
	}

	endpoint := fmt.Sprintf(
		"%s/api/v1/canvases/%s/node-executions/%s/runner-live-logs/session",
		strings.TrimRight(baseURL, "/"),
		url.PathEscape(canvasID),
		url.PathEscape(executionID),
	)

	request, err := http.NewRequestWithContext(ctx.Context, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	if authorization := strings.TrimSpace(config.DefaultHeader["Authorization"]); authorization != "" {
		request.Header.Set("Authorization", authorization)
	}

	response, err := logHTTPClient(config).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("create runner log session for execution %s: %s", executionID, responseErrorMessage(response.Status, body))
	}

	var session runneraction.LiveLogSession
	if err := json.Unmarshal(body, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func responseErrorMessage(status string, body []byte) string {
	message := strings.TrimSpace(string(body))
	if message != "" {
		return message
	}
	return status
}

func normalizedLogLimit(limit *int64) int {
	if limit == nil || *limit <= 0 {
		return runneraction.DefaultLiveLogRecordLimit
	}
	if *limit > int64(runneraction.MaxLiveLogRecordLimit) {
		return runneraction.MaxLiveLogRecordLimit
	}
	return int(*limit)
}

func logHTTPClient(config *openapi_client.Configuration) *http.Client {
	if config != nil && config.HTTPClient != nil {
		return config.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func renderRunnerLogsText(stdout io.Writer, outputs []runnerLogOutput) error {
	for i, output := range outputs {
		if i > 0 {
			_, _ = fmt.Fprintln(stdout)
		}
		if err := renderRunnerLogHeader(stdout, output); err != nil {
			return err
		}
		if output.Error != "" {
			if _, err := fmt.Fprintf(stdout, "Error: %s\n", output.Error); err != nil {
				return err
			}
			continue
		}
		for _, record := range output.Records {
			if err := renderRunnerLogRecord(stdout, record); err != nil {
				return err
			}
		}
		if output.Truncated {
			if _, err := fmt.Fprintln(stdout, "... truncated"); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderRunnerLogHeader(stdout io.Writer, output runnerLogOutput) error {
	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintf(writer, "Execution\t%s\n", output.ExecutionID)
	if output.NodeID != "" {
		_, _ = fmt.Fprintf(writer, "Node\t%s\n", output.NodeID)
	}
	_, _ = fmt.Fprintln(writer, "Logs")
	return writer.Flush()
}

func renderRunnerLogRecord(stdout io.Writer, record runneraction.LiveLogRecord) error {
	switch record.Type {
	case "line":
		_, err := fmt.Fprintln(stdout, record.Text)
		return err
	case "error":
		_, err := fmt.Fprintf(stdout, "ERROR: %s\n", record.Message)
		return err
	case "cmd_start":
		_, err := fmt.Fprintf(stdout, "$ %s\n", record.Text)
		return err
	case "cmd_end":
		_, err := fmt.Fprintf(stdout, "# command %s (%dms)\n", record.Status, int64Value(record.DurationMS))
		return err
	default:
		return nil
	}
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
