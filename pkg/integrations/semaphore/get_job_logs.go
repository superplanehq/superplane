package semaphore

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const GetJobLogsPayloadType = "semaphore.job.logs"
const GetJobLogsSuccessChannel = "success"
const GetJobLogsDefaultLimit = 1000
const GetJobLogsMaxLimit = 5000

type GetJobLogs struct{}

type GetJobLogsSpec struct {
	JobID string `json:"jobId" mapstructure:"jobId"`
	Limit int    `json:"limit" mapstructure:"limit"`
}

type GetJobLogsOutput struct {
	JobID    string       `json:"jobId"`
	JobName  string       `json:"jobName"`
	State    string       `json:"state"`
	Result   string       `json:"result"`
	Logs     string       `json:"logs"`
	LogLines []string     `json:"logLines"`
	Metadata *JobMetadata `json:"metadata"`
}

func (g *GetJobLogs) Name() string {
	return "semaphore.getJobLogs"
}

func (g *GetJobLogs) Label() string {
	return "Get Job Logs"
}

func (g *GetJobLogs) Description() string {
	return "Fetch logs for a Semaphore job"
}

func (g *GetJobLogs) Documentation() string {
	return `The Get Job Logs component fetches the log output for a Semaphore job by job ID.

## Use Cases

- **Failure analysis**: When a pipeline fails, fetch the failing job's logs and send them in a Slack or PagerDuty notification.
- **Debug automation**: Attach job logs to a ticket or runbook step for debugging (e.g. "last 500 lines of job X").
- **Log parsing**: Parse job logs in a workflow (e.g. extract test summary or error lines) and branch or report based on content.
- **Audit trail**: Store job logs for compliance or audit purposes.

## How It Works

1. Fetches the job details from Semaphore API to get job metadata
2. Fetches the job logs from Semaphore API
3. Applies the configured line limit (last N lines)
4. Emits the logs and metadata on the success channel

## Configuration

- **Job ID** (required): The Semaphore job ID. Can be obtained from pipeline/block event data (e.g., ` + "`data.blocks[].jobs[].id`" + `), or from Run Workflow / Get Pipeline output. Accepts expressions.
- **Limit** (optional): Maximum number of log lines to return. Default is 1000, maximum is 5000. Returns the last N lines.

## Output

Single output channel that emits:
- ` + "`jobId`" + `: The job ID
- ` + "`jobName`" + `: The job name
- ` + "`state`" + `: Job state (e.g., "finished")
- ` + "`result`" + `: Job result (e.g., "passed", "failed", "stopped")
- ` + "`logs`" + `: Raw log content as a single string
- ` + "`logLines`" + `: Log content as an array of lines
- ` + "`metadata`" + `: Job metadata including timestamps

## Notes

- The component retrieves all available logs and applies the limit locally
- If the job doesn't exist or logs are unavailable, an error is returned
- Logs are returned in chronological order (oldest first)`
}

func (g *GetJobLogs) Icon() string {
	return "file-text"
}

func (g *GetJobLogs) Color() string {
	return "gray"
}

func (g *GetJobLogs) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{
			Name:  GetJobLogsSuccessChannel,
			Label: "Success",
		},
	}
}

func (g *GetJobLogs) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "jobId",
			Label:       "Job ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g. a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Description: "The Semaphore job ID. Can be obtained from pipeline event data or Run Workflow output.",
		},
		{
			Name:        "limit",
			Label:       "Line Limit",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     GetJobLogsDefaultLimit,
			Placeholder: fmt.Sprintf("Default: %d, Max: %d", GetJobLogsDefaultLimit, GetJobLogsMaxLimit),
			Description: "Maximum number of log lines to return (returns last N lines).",
		},
	}
}

func (g *GetJobLogs) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetJobLogs) Setup(ctx core.SetupContext) error {
	spec := GetJobLogsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Validate limit if provided
	if spec.Limit < 0 {
		return fmt.Errorf("limit must be a positive number")
	}

	if spec.Limit > GetJobLogsMaxLimit {
		return fmt.Errorf("limit cannot exceed %d", GetJobLogsMaxLimit)
	}

	return nil
}

func (g *GetJobLogs) Execute(ctx core.ExecutionContext) error {
	spec := GetJobLogsSpec{}
	err := mapstructure.Decode(ctx.Configuration, &spec)
	if err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.JobID == "" {
		return fmt.Errorf("job ID is required")
	}

	// Apply default limit if not specified
	limit := spec.Limit
	if limit == 0 {
		limit = GetJobLogsDefaultLimit
	}
	if limit > GetJobLogsMaxLimit {
		limit = GetJobLogsMaxLimit
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("error creating client: %w", err)
	}

	// Fetch job details
	ctx.Logger.Infof("Fetching job details for job=%s", spec.JobID)
	job, err := client.GetJob(spec.JobID)
	if err != nil {
		return fmt.Errorf("error fetching job %s: %w", spec.JobID, err)
	}

	// Fetch job logs
	ctx.Logger.Infof("Fetching logs for job=%s", spec.JobID)
	logs, err := client.GetJobLogs(spec.JobID)
	if err != nil {
		return fmt.Errorf("error fetching logs for job %s: %w", spec.JobID, err)
	}

	// Process log events into lines
	var logLines []string
	for _, event := range logs.Events {
		if event.Output != "" {
			// Split output by newlines in case single event contains multiple lines
			lines := strings.Split(event.Output, "\n")
			for _, line := range lines {
				if line != "" {
					logLines = append(logLines, line)
				}
			}
		}
	}

	// Apply limit (take last N lines)
	if len(logLines) > limit {
		logLines = logLines[len(logLines)-limit:]
	}

	// Join logs into single string
	logsText := strings.Join(logLines, "\n")

	ctx.Logger.Infof("Retrieved %d log lines for job=%s (limit=%d)", len(logLines), spec.JobID, limit)

	output := GetJobLogsOutput{
		JobID:    spec.JobID,
		JobName:  job.Metadata.Name,
		State:    job.Status.State,
		Result:   job.Status.Result,
		Logs:     logsText,
		LogLines: logLines,
		Metadata: &job.Metadata,
	}

	// Store metadata for reference
	ctx.Metadata.Set(map[string]any{
		"jobId":     spec.JobID,
		"jobName":   job.Metadata.Name,
		"lineCount": len(logLines),
	})

	return ctx.Requests.Emit(GetJobLogsSuccessChannel, GetJobLogsPayloadType, []any{output})
}

func (g *GetJobLogs) Cancel(ctx core.ExecutionContext) error {
	// Nothing to cancel - this is a synchronous operation
	return nil
}

func (g *GetJobLogs) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetJobLogs) HandleAction(ctx core.ActionContext) error {
	return fmt.Errorf("no actions available for GetJobLogs")
}

func (g *GetJobLogs) Cleanup(ctx core.SetupContext) error {
	return nil
}
