package semaphore

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

type GetJobLogs struct{}

type GetJobLogsConfiguration struct {
	JobID string `mapstructure:"jobId"`
	Limit *int   `mapstructure:"limit,omitempty"`
}

func (c *GetJobLogs) Name() string {
	return "semaphore.getJobLogs"
}

func (c *GetJobLogs) Label() string {
	return "Get Job Logs"
}

func (c *GetJobLogs) Description() string {
	return "Get Semaphore job log output"
}

func (c *GetJobLogs) Documentation() string {
	return `The Get Job Logs component retrieves the log output for a Semaphore job by job ID.

## Use Cases

- **Failure Analysis**: When a pipeline fails, fetch the failing job's logs and send them in a Slack or PagerDuty notification
- **Debugging**: Attach job logs to a ticket or runbook step for debugging
- **Log Parsing**: Parse job logs in a workflow to extract test summaries or error lines and branch based on content

## Configuration

- **Job ID**: The Semaphore job ID (e.g., from pipeline/block event data or from Run Workflow output). Accepts expressions.
- **Limit**: Optional maximum number of log lines to return. Default returns all logs, max 1000 lines.

## Output

Returns job log content including:
- Event types (job_started, cmd_started, cmd_output, cmd_finished, job_finished)
- Timestamps for each event
- Command output text
- Exit codes and results

## Notes

- Job IDs can be found in pipeline event data under blocks[].jobs[].id
- Logs are returned as structured events that can be parsed or formatted downstream
- Use with On Pipeline Done or Run Workflow components to react to job failures`
}

func (c *GetJobLogs) Icon() string {
	return "file-text"
}

func (c *GetJobLogs) Color() string {
	return "gray"
}

func (c *GetJobLogs) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetJobLogs) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "jobId",
			Label:       "Job ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., job-uuid or {{$.data.blocks[0].jobs[0].id}}",
			Description: "The Semaphore job ID to fetch logs for. Supports template expressions.",
		},
		{
			Name:        "limit",
			Label:       "Line Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Placeholder: "e.g., 500",
			Description: "Maximum number of log output lines to return. Default returns all, max 1000.",
		},
	}
}

func (c *GetJobLogs) Setup(ctx core.SetupContext) error {
	return nil
}

func (c *GetJobLogs) Execute(ctx core.ExecutionContext) error {
	var config GetJobLogsConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Validate required fields
	if config.JobID == "" {
		return fmt.Errorf("job ID is required")
	}

	// Validate limit if provided
	maxLimit := 1000
	if config.Limit != nil && *config.Limit > maxLimit {
		return fmt.Errorf("limit cannot exceed %d lines", maxLimit)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to initialize Semaphore client: %w", err)
	}

	// Fetch job logs
	logs, err := client.GetJobLogs(config.JobID)
	if err != nil {
		return fmt.Errorf("failed to get job logs: %w", err)
	}

	// Build output with optional line limiting
	output := c.buildLogsOutput(logs, config.Limit)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"semaphore.jobLogs",
		[]any{output},
	)
}

func (c *GetJobLogs) buildLogsOutput(logs *JobLogsResponse, limit *int) map[string]any {
	output := map[string]any{
		"events": logs.Events,
	}

	// Extract just the output lines for convenience
	var outputLines []string
	for _, event := range logs.Events {
		if event.Event == "cmd_output" && event.Output != "" {
			outputLines = append(outputLines, event.Output)
		}
	}

	// Apply limit if specified
	if limit != nil && *limit > 0 && len(outputLines) > *limit {
		outputLines = outputLines[len(outputLines)-*limit:]
	}

	output["output"] = strings.Join(outputLines, "")
	output["lineCount"] = len(outputLines)

	// Extract job result if available
	for _, event := range logs.Events {
		if event.Event == "job_finished" {
			output["result"] = event.Result
			break
		}
	}

	return output
}

func (c *GetJobLogs) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetJobLogs) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 200, nil
}

func (c *GetJobLogs) Actions() []core.Action {
	return []core.Action{}
}

func (c *GetJobLogs) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *GetJobLogs) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetJobLogs) Cleanup(ctx core.SetupContext) error {
	return nil
}
