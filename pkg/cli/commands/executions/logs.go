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
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type LogsCommand struct {
	CanvasID    *string
	ExecutionID *string
	RunID       *string
	NodeID      *string
	Limit       *int64
}

type runnerLogRecord struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	Message    string `json:"message,omitempty"`
	Index      *int   `json:"index,omitempty"`
	Status     string `json:"status,omitempty"`
	DurationMS *int64 `json:"duration_ms,omitempty"`
	StartedAt  *int64 `json:"started_at,omitempty"`
}

type runnerLogsResponse struct {
	CanvasID     string            `json:"canvas_id"`
	ExecutionID  string            `json:"execution_id"`
	BrokerTaskID string            `json:"broker_task_id"`
	Count        int               `json:"count"`
	Truncated    bool              `json:"truncated,omitempty"`
	Records      []runnerLogRecord `json:"records"`
}

type runnerLogTarget struct {
	ExecutionID string
	NodeID      string
}

type runnerLogOutput struct {
	CanvasID     string            `json:"canvas_id"`
	ExecutionID  string            `json:"execution_id"`
	NodeID       string            `json:"node_id,omitempty"`
	BrokerTaskID string            `json:"broker_task_id,omitempty"`
	Count        int               `json:"count"`
	Truncated    bool              `json:"truncated,omitempty"`
	Records      []runnerLogRecord `json:"records,omitempty"`
	Error        string            `json:"error,omitempty"`
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

	run := response.GetRun()
	executions := run.GetExecutions()
	targets := make([]runnerLogTarget, 0, len(executions))
	for _, execution := range executions {
		if strings.TrimSpace(*c.NodeID) != "" && execution.GetNodeId() != strings.TrimSpace(*c.NodeID) {
			continue
		}
		targets = append(targets, runnerLogTarget{
			ExecutionID: execution.GetId(),
			NodeID:      execution.GetNodeId(),
		})
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no executions found for log target")
	}
	return targets, nil
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

	response, err := c.fetchExecutionLogs(ctx, canvasID, target.ExecutionID)
	if err != nil {
		output.Error = err.Error()
		return output
	}

	output.BrokerTaskID = response.BrokerTaskID
	output.Count = response.Count
	output.Truncated = response.Truncated
	output.Records = response.Records
	return output
}

func (c *LogsCommand) fetchExecutionLogs(ctx core.CommandContext, canvasID, executionID string) (*runnerLogsResponse, error) {
	config := ctx.API.GetConfig()
	if config == nil {
		return nil, fmt.Errorf("api client config is required")
	}

	baseURL, err := config.ServerURLWithContext(ctx.Context, "CanvasNodeExecutionAPIService.CanvasesCancelExecution")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("api_url is required")
	}

	endpoint := fmt.Sprintf(
		"%s/api/v1/canvases/%s/node-executions/%s/runner-live-logs?limit=%d",
		strings.TrimRight(baseURL, "/"),
		url.PathEscape(canvasID),
		url.PathEscape(executionID),
		normalizedLogLimit(c.Limit),
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
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return nil, fmt.Errorf("%s", message)
	}

	var payload runnerLogsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func normalizedLogLimit(limit *int64) int64 {
	if limit == nil || *limit <= 0 {
		return 200
	}
	return *limit
}

func logHTTPClient(config *openapi_client.Configuration) *http.Client {
	if config.HTTPClient != nil {
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
			_, err := fmt.Fprintf(stdout, "Error: %s\n", output.Error)
			return err
		}
		for _, record := range output.Records {
			if err := renderRunnerLogRecord(stdout, record); err != nil {
				return err
			}
		}
		if output.Truncated {
			_, err := fmt.Fprintln(stdout, "... truncated")
			return err
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
	if output.BrokerTaskID != "" {
		_, _ = fmt.Fprintf(writer, "Broker task\t%s\n", output.BrokerTaskID)
	}
	_, _ = fmt.Fprintln(writer, "Logs")
	return writer.Flush()
}

func renderRunnerLogRecord(stdout io.Writer, record runnerLogRecord) error {
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
